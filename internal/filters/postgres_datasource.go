package filters

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	_ "github.com/jackc/pgx/v5/stdlib"
)

const postgresResponseSchemaTable = "public.testdataset_response_schemas"

// QueryPostgresDataSource loads datasource rows from Postgres and runs the shared filter pipeline.
func QueryPostgresDataSource(
	req FilterRequest,
	dsn string,
	dataTable string,
	schemaTable string,
	maxItems int,
) (CompiledFilter, AllowedFieldResponse, DataSetResponse, error) {
	return QueryPostgresDataSourceWithSeed(req, dsn, dataTable, schemaTable, maxItems, "", 0)
}

// QueryPostgresDataSourceWithSeed behaves like QueryPostgresDataSource but allows deterministic
// shuffling when randomSeedGUID is set (with optional deterministic offset).
func QueryPostgresDataSourceWithSeed(
	req FilterRequest,
	dsn string,
	dataTable string,
	schemaTable string,
	maxItems int,
	randomSeedGUID string,
	randomSeedOffset int,
) (CompiledFilter, AllowedFieldResponse, DataSetResponse, error) {
	if err := validateRequestEnvelope(req); err != nil {
		return CompiledFilter{}, AllowedFieldResponse{}, DataSetResponse{}, err
	}
	if strings.TrimSpace(dsn) == "" {
		return CompiledFilter{}, AllowedFieldResponse{}, DataSetResponse{}, errors.New("postgres dsn is required")
	}
	if strings.TrimSpace(dataTable) == "" {
		dataTable = "public.data_items"
	}
	if strings.TrimSpace(schemaTable) == "" {
		schemaTable = postgresResponseSchemaTable
	}
	if !isSafeTableIdentifier(dataTable) {
		return CompiledFilter{}, AllowedFieldResponse{}, DataSetResponse{}, fmt.Errorf("unsafe table name: %q", dataTable)
	}
	if !isSafeTableIdentifier(schemaTable) {
		return CompiledFilter{}, AllowedFieldResponse{}, DataSetResponse{}, fmt.Errorf("unsafe schema metadata table name: %q", schemaTable)
	}
	if maxItems < 0 {
		maxItems = 0
	}

	ds, rows, schemaMeta, err := loadPostgresDataSource(req, dsn, dataTable, schemaTable)
	if err != nil {
		return CompiledFilter{}, AllowedFieldResponse{}, DataSetResponse{}, err
	}
	return queryDataRows(req, ds, rows, "postgres", maxItems, randomSeedGUID, randomSeedOffset, schemaMeta)
}

func loadPostgresDataSource(
	req FilterRequest,
	dsn string,
	dataTable string,
	schemaTable string,
) (DataSourceDefinition, []map[string]interface{}, *DataSetSchemaMetadata, error) {
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return DataSourceDefinition{}, nil, nil, fmt.Errorf("open postgres db: %w", err)
	}
	defer db.Close()

	schemaMeta, err := loadPostgresDataSetSchemaMetadata(context.Background(), db, req, schemaTable)
	if err != nil {
		return DataSourceDefinition{}, nil, nil, err
	}

	query := fmt.Sprintf(
		`SELECT "JsonData"::text FROM %s WHERE "DataSourceUuid" = $1 AND "DataSourceName" = $2`,
		quoteQualifiedIdentifier(dataTable),
	)
	rows, err := db.QueryContext(context.Background(), query, req.DataSourceUUID, req.DataSourceName)
	if err != nil {
		return DataSourceDefinition{}, nil, nil, fmt.Errorf("query postgres datasource rows: %w", err)
	}
	defer rows.Close()

	rawRows := make([]map[string]interface{}, 0)
	for rows.Next() {
		var rawJSON string
		if err := rows.Scan(&rawJSON); err != nil {
			return DataSourceDefinition{}, nil, nil, fmt.Errorf("scan postgres row: %w", err)
		}

		var payload map[string]interface{}
		if err := json.Unmarshal([]byte(rawJSON), &payload); err != nil {
			return DataSourceDefinition{}, nil, nil, fmt.Errorf("unmarshal JsonData: %w", err)
		}
		rawRows = append(rawRows, payload)
	}
	if err := rows.Err(); err != nil {
		return DataSourceDefinition{}, nil, nil, fmt.Errorf("iterate postgres rows: %w", err)
	}
	if len(rawRows) == 0 {
		return DataSourceDefinition{}, nil, nil, fmt.Errorf("no data rows found for datasource %q (%s)", req.DataSourceName, req.DataSourceUUID)
	}

	ds, typedRows, err := loadPostgresDataSourceFromSchema(req.DataSourceUUID, schemaMeta, rawRows)
	if err == nil {
		return ds, typedRows, schemaMeta, nil
	}

	// Fall back to the legacy inference path when schema-driven loading is unavailable.
	return loadPostgresDataSourceByInference(req.DataSourceUUID, rawRows, schemaMeta)
}

func loadPostgresDataSourceFromSchema(
	dataSourceUUID string,
	schemaMeta *DataSetSchemaMetadata,
	rawRows []map[string]interface{},
) (DataSourceDefinition, []map[string]interface{}, error) {
	catalog, err := schemaCatalogForDataSource(schemaMeta)
	if err != nil {
		return DataSourceDefinition{}, nil, err
	}

	ds := schemaDataSourceDefinition(catalog, dataSourceUUID)
	fields := ds.Fields
	normalizedRows := normalizeRowsToCanonical(rawRows, catalog)
	typedRows := make([]map[string]interface{}, 0, len(normalizedRows))

	for idx, rawRow := range normalizedRows {
		row := make(map[string]interface{}, len(catalog.Order))
		for _, field := range catalog.Order {
			fieldDef, ok := fields[field]
			if !ok {
				continue
			}
			val, err := coerceRawValue(rawRow[field], fieldDef.FieldType)
			if err != nil {
				return DataSourceDefinition{}, nil, fmt.Errorf("postgres row %d field %q: %w", idx+1, field, err)
			}
			row[field] = val
		}
		typedRows = append(typedRows, row)
	}

	return ds, typedRows, nil
}

func loadPostgresDataSourceByInference(
	dataSourceUUID string,
	rawRows []map[string]interface{},
	schemaMeta *DataSetSchemaMetadata,
) (DataSourceDefinition, []map[string]interface{}, *DataSetSchemaMetadata, error) {
	fieldOrder := collectFieldOrder(rawRows)
	columnValues := make(map[string][]string, len(fieldOrder))
	for _, row := range rawRows {
		for _, field := range fieldOrder {
			columnValues[field] = append(columnValues[field], rawValueToStringForInference(row[field]))
		}
	}

	fields := make(map[string]FieldDefinition, len(fieldOrder))
	for _, field := range fieldOrder {
		fieldType := inferFieldType(columnValues[field])
		fields[field] = FieldDefinition{
			FieldType:          fieldType,
			Nullable:           hasNullValue(columnValues[field]),
			SupportedOperators: supportedOperatorsForType(fieldType),
			Description:        fmt.Sprintf("Inferred from Postgres JsonData field %q.", field),
		}
	}

	typedRows := make([]map[string]interface{}, 0, len(rawRows))
	for idx, rawRow := range rawRows {
		row := make(map[string]interface{}, len(fieldOrder))
		for _, field := range fieldOrder {
			fieldDef := fields[field]
			val, err := coerceRawValue(rawRow[field], fieldDef.FieldType)
			if err != nil {
				return DataSourceDefinition{}, nil, nil, fmt.Errorf("postgres row %d field %q: %w", idx+1, field, err)
			}
			row[field] = val
		}
		typedRows = append(typedRows, row)
	}

	return DataSourceDefinition{
		UUID:   dataSourceUUID,
		Fields: fields,
	}, typedRows, schemaMeta, nil
}

func loadPostgresDataSetSchemaMetadata(
	ctx context.Context,
	db *sql.DB,
	req FilterRequest,
	schemaTable string,
) (*DataSetSchemaMetadata, error) {
	query := fmt.Sprintf(
		`SELECT "TestDataSourceName", "TestDataSourceUuid", "JsonSchemaName", "JsonSchema"::text, "UpdatedDateTime"::text
FROM %s
WHERE lower("TestDataSourceUuid"::text) = lower($1) AND "TestDataSourceName" = $2
ORDER BY "UpdatedDateTime" DESC
LIMIT 1`,
		quoteQualifiedIdentifier(schemaTable),
	)

	var (
		meta       DataSetSchemaMetadata
		jsonSchema string
	)
	err := db.QueryRowContext(ctx, query, req.DataSourceUUID, req.DataSourceName).Scan(
		&meta.TestDataSourceName,
		&meta.TestDataSourceUUID,
		&meta.JsonSchemaName,
		&jsonSchema,
		&meta.UpdatedDateTime,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		lowerErr := strings.ToLower(err.Error())
		if strings.Contains(lowerErr, "does not exist") || strings.Contains(lowerErr, "undefined table") || strings.Contains(lowerErr, "relation") {
			return nil, nil
		}
		return nil, fmt.Errorf("query response schema metadata: %w", err)
	}
	if !json.Valid([]byte(jsonSchema)) {
		return nil, fmt.Errorf("invalid json schema in %s for datasource %q (%s)", schemaTable, req.DataSourceName, req.DataSourceUUID)
	}

	meta.JsonSchemaName = CanonicalResponseSchemaName(meta.JsonSchemaName)
	meta.JsonSchema = json.RawMessage(jsonSchema)
	return &meta, nil
}

func quoteQualifiedIdentifier(raw string) string {
	parts := strings.Split(raw, ".")
	quoted := make([]string, 0, len(parts))
	for _, part := range parts {
		quoted = append(quoted, `"`+part+`"`)
	}
	return strings.Join(quoted, ".")
}
