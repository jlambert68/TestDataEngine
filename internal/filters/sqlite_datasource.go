package filters

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"

	_ "modernc.org/sqlite"
)

const responseSchemaTable = "main.testdataset_response_schemas"

// QuerySQLiteDataSource loads datasource rows from SQLite and runs the shared filter pipeline.
func QuerySQLiteDataSource(
	req FilterRequest,
	dbPath string,
	tableName string,
	maxItems int,
) (CompiledFilter, AllowedFieldResponse, DataSetResponse, error) {
	return QuerySQLiteDataSourceWithSeed(req, dbPath, tableName, maxItems, "")
}

// QuerySQLiteDataSourceWithSeed behaves like QuerySQLiteDataSource but allows deterministic
// shuffling when randomSeedGUID is set.
func QuerySQLiteDataSourceWithSeed(
	req FilterRequest,
	dbPath string,
	tableName string,
	maxItems int,
	randomSeedGUID string,
) (CompiledFilter, AllowedFieldResponse, DataSetResponse, error) {
	if err := validateRequestEnvelope(req); err != nil {
		return CompiledFilter{}, AllowedFieldResponse{}, DataSetResponse{}, err
	}
	if strings.TrimSpace(dbPath) == "" {
		return CompiledFilter{}, AllowedFieldResponse{}, DataSetResponse{}, errors.New("db path is required")
	}
	if strings.TrimSpace(tableName) == "" {
		tableName = "main.data_items"
	}
	if !isSafeTableIdentifier(tableName) {
		return CompiledFilter{}, AllowedFieldResponse{}, DataSetResponse{}, fmt.Errorf("unsafe table name: %q", tableName)
	}
	if maxItems < 0 {
		maxItems = 0
	}

	ds, rows, schemaMeta, err := loadSQLiteDataSource(req, dbPath, tableName)
	if err != nil {
		return CompiledFilter{}, AllowedFieldResponse{}, DataSetResponse{}, err
	}
	return queryDataRows(req, ds, rows, "sqlite", maxItems, randomSeedGUID, schemaMeta)
}

// loadSQLiteDataSource fetches JsonData rows and converts them to typed rows.
func loadSQLiteDataSource(
	req FilterRequest,
	dbPath string,
	tableName string,
) (DataSourceDefinition, []map[string]interface{}, *DataSetSchemaMetadata, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return DataSourceDefinition{}, nil, nil, fmt.Errorf("open sqlite db: %w", err)
	}
	defer db.Close()

	schemaMeta, err := loadDataSetSchemaMetadata(context.Background(), db, req)
	if err != nil {
		return DataSourceDefinition{}, nil, nil, err
	}

	query := fmt.Sprintf(
		`SELECT JsonData FROM %s WHERE DataSourceUuid = ? AND DataSourceName = ?`,
		tableName,
	)
	rows, err := db.QueryContext(context.Background(), query, req.DataSourceUUID, req.DataSourceName)
	if err != nil {
		return DataSourceDefinition{}, nil, nil, fmt.Errorf("query sqlite datasource rows: %w", err)
	}
	defer rows.Close()

	rawRows := make([]map[string]interface{}, 0)
	for rows.Next() {
		var rawJSON string
		if err := rows.Scan(&rawJSON); err != nil {
			return DataSourceDefinition{}, nil, nil, fmt.Errorf("scan sqlite row: %w", err)
		}

		var payload map[string]interface{}
		if err := json.Unmarshal([]byte(rawJSON), &payload); err != nil {
			return DataSourceDefinition{}, nil, nil, fmt.Errorf("unmarshal JsonData: %w", err)
		}
		rawRows = append(rawRows, payload)
	}
	if err := rows.Err(); err != nil {
		return DataSourceDefinition{}, nil, nil, fmt.Errorf("iterate sqlite rows: %w", err)
	}
	if len(rawRows) == 0 {
		return DataSourceDefinition{}, nil, nil, fmt.Errorf("no data rows found for datasource %q (%s)", req.DataSourceName, req.DataSourceUUID)
	}

	ds, typedRows, err := loadSQLiteDataSourceFromSchema(req.DataSourceUUID, schemaMeta, rawRows)
	if err == nil {
		return ds, typedRows, schemaMeta, nil
	}

	// Fall back to the legacy inference path when schema-driven loading is unavailable.
	return loadSQLiteDataSourceByInference(req.DataSourceUUID, rawRows, schemaMeta)
}

func loadSQLiteDataSourceFromSchema(
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
				return DataSourceDefinition{}, nil, fmt.Errorf("sqlite row %d field %q: %w", idx+1, field, err)
			}
			row[field] = val
		}
		typedRows = append(typedRows, row)
	}

	return ds, typedRows, nil
}

func loadSQLiteDataSourceByInference(
	dataSourceUUID string,
	rawRows []map[string]interface{},
	schemaMeta *DataSetSchemaMetadata,
) (DataSourceDefinition, []map[string]interface{}, *DataSetSchemaMetadata, error) {
	// Build inferred field definitions from all discovered JSON keys.
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
			Description:        fmt.Sprintf("Inferred from SQLite JsonData field %q.", field),
		}
	}

	typedRows := make([]map[string]interface{}, 0, len(rawRows))
	for idx, rawRow := range rawRows {
		row := make(map[string]interface{}, len(fieldOrder))
		for _, field := range fieldOrder {
			fieldDef := fields[field]
			val, err := coerceRawValue(rawRow[field], fieldDef.FieldType)
			if err != nil {
				return DataSourceDefinition{}, nil, nil, fmt.Errorf("sqlite row %d field %q: %w", idx+1, field, err)
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

// loadDataSetSchemaMetadata fetches response-schema metadata for the requested datasource.
func loadDataSetSchemaMetadata(ctx context.Context, db *sql.DB, req FilterRequest) (*DataSetSchemaMetadata, error) {
	query := fmt.Sprintf(
		`SELECT TestDataSourceName, TestDataSourceUuid, JsonSchemaName, JsonSchema, UpdatedDateTime
FROM %s
WHERE lower(TestDataSourceUuid) = lower(?) AND TestDataSourceName = ?
ORDER BY UpdatedDateTime DESC
LIMIT 1`,
		responseSchemaTable,
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
		// Keep backward compatibility with SQLite DBs that predate the schema-metadata table.
		if strings.Contains(strings.ToLower(err.Error()), "no such table") {
			return nil, nil
		}
		return nil, fmt.Errorf("query response schema metadata: %w", err)
	}
	if !json.Valid([]byte(jsonSchema)) {
		return nil, fmt.Errorf("invalid json schema in %s for datasource %q (%s)", responseSchemaTable, req.DataSourceName, req.DataSourceUUID)
	}

	meta.JsonSchemaName = CanonicalResponseSchemaName(meta.JsonSchemaName)
	meta.JsonSchema = json.RawMessage(jsonSchema)
	return &meta, nil
}

// collectFieldOrder returns a stable sorted list of field names across all rows.
func collectFieldOrder(rows []map[string]interface{}) []string {
	order := make([]string, 0)
	seen := make(map[string]struct{})
	for _, row := range rows {
		for field := range row {
			if _, ok := seen[field]; ok {
				continue
			}
			seen[field] = struct{}{}
			order = append(order, field)
		}
	}
	sortStrings(order)
	return order
}

// sortStrings provides deterministic ordering without extra dependencies.
func sortStrings(items []string) {
	for i := 0; i < len(items)-1; i++ {
		for j := i + 1; j < len(items); j++ {
			if items[j] < items[i] {
				items[i], items[j] = items[j], items[i]
			}
		}
	}
}

// rawValueToStringForInference converts JSON values to strings used by CSV-type inference helpers.
func rawValueToStringForInference(v interface{}) string {
	if v == nil {
		return "NULL"
	}
	switch t := v.(type) {
	case string:
		return t
	case bool:
		if t {
			return "true"
		}
		return "false"
	case float64:
		return strconv.FormatFloat(t, 'f', -1, 64)
	case float32:
		return strconv.FormatFloat(float64(t), 'f', -1, 64)
	case int:
		return strconv.Itoa(t)
	case int8:
		return strconv.FormatInt(int64(t), 10)
	case int16:
		return strconv.FormatInt(int64(t), 10)
	case int32:
		return strconv.FormatInt(int64(t), 10)
	case int64:
		return strconv.FormatInt(t, 10)
	case uint:
		return strconv.FormatUint(uint64(t), 10)
	case uint8:
		return strconv.FormatUint(uint64(t), 10)
	case uint16:
		return strconv.FormatUint(uint64(t), 10)
	case uint32:
		return strconv.FormatUint(uint64(t), 10)
	case uint64:
		return strconv.FormatUint(t, 10)
	default:
		return fmt.Sprintf("%v", v)
	}
}

// coerceRawValue maps raw JSON values into the inferred field type.
func coerceRawValue(v interface{}, fieldType string) (interface{}, error) {
	if v == nil {
		return nil, nil
	}
	raw := rawValueToStringForInference(v)
	return parseCSVValue(raw, fieldType)
}

// isSafeTableIdentifier limits dynamic table names to safe identifier characters.
func isSafeTableIdentifier(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if (r >= 'a' && r <= 'z') ||
			(r >= 'A' && r <= 'Z') ||
			(r >= '0' && r <= '9') ||
			r == '_' || r == '.' {
			continue
		}
		return false
	}
	return true
}
