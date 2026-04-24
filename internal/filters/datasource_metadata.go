package filters

import (
	"context"
	"database/sql"
	"fmt"
	"sort"
	"strings"
)

// FacetValueCount represents one distinct field value and the number of rows carrying it.
type FacetValueCount struct {
	Value  interface{}
	Count  int
	IsNull bool
}

// DescribeCSVDataSource returns inferred field metadata for a CSV datasource.
func DescribeCSVDataSource(req FilterRequest, csvPath string) (AllowedFieldResponse, error) {
	if err := validateMetadataRequest(req); err != nil {
		return AllowedFieldResponse{}, err
	}
	ds, _, err := loadCSVDataSource(req.DataSourceUUID, csvPath)
	if err != nil {
		return AllowedFieldResponse{}, err
	}
	return allowedFieldsForDataSource(req, ds)
}

// DescribeSQLiteDataSource returns inferred field metadata for a SQLite datasource.
func DescribeSQLiteDataSource(req FilterRequest, dbPath string, tableName string) (AllowedFieldResponse, error) {
	if err := validateMetadataRequest(req); err != nil {
		return AllowedFieldResponse{}, err
	}
	if schemaResp, err := describeDataSourceFromSQLiteSchema(req, dbPath); err == nil {
		return schemaResp, nil
	}
	if strings.TrimSpace(tableName) == "" {
		tableName = "main.data_items"
	}
	ds, _, _, err := loadSQLiteDataSource(req, dbPath, tableName)
	if err != nil {
		return AllowedFieldResponse{}, err
	}
	return allowedFieldsForDataSource(req, ds)
}

// DescribePostgresDataSource returns inferred field metadata for a Postgres datasource.
func DescribePostgresDataSource(req FilterRequest, dsn string, dataTable string, schemaTable string) (AllowedFieldResponse, error) {
	if err := validateMetadataRequest(req); err != nil {
		return AllowedFieldResponse{}, err
	}
	if schemaResp, err := describeDataSourceFromPostgresSchema(req, dsn, schemaTable); err == nil {
		return schemaResp, nil
	}
	if strings.TrimSpace(dataTable) == "" {
		dataTable = "public.data_items"
	}
	if strings.TrimSpace(schemaTable) == "" {
		schemaTable = postgresResponseSchemaTable
	}
	ds, _, _, err := loadPostgresDataSource(req, dsn, dataTable, schemaTable)
	if err != nil {
		return AllowedFieldResponse{}, err
	}
	return allowedFieldsForDataSource(req, ds)
}

// FacetCSVDataSource returns distinct value counts for one CSV field.
func FacetCSVDataSource(req FilterRequest, csvPath string, field string, limit int, q string) ([]FacetValueCount, bool, error) {
	if err := validateMetadataRequest(req); err != nil {
		return nil, false, err
	}
	ds, rows, err := loadCSVDataSource(req.DataSourceUUID, csvPath)
	if err != nil {
		return nil, false, err
	}
	return facetValuesForRows(ds, rows, field, limit, q)
}

// FacetSQLiteDataSource returns distinct value counts for one SQLite field.
func FacetSQLiteDataSource(req FilterRequest, dbPath string, tableName string, field string, limit int, q string) ([]FacetValueCount, bool, error) {
	if err := validateMetadataRequest(req); err != nil {
		return nil, false, err
	}
	if strings.TrimSpace(tableName) == "" {
		tableName = "main.data_items"
	}
	ds, rows, _, err := loadSQLiteDataSource(req, dbPath, tableName)
	if err != nil {
		return nil, false, err
	}
	return facetValuesForRows(ds, rows, field, limit, q)
}

// FacetPostgresDataSource returns distinct value counts for one Postgres field.
func FacetPostgresDataSource(req FilterRequest, dsn string, dataTable string, schemaTable string, field string, limit int, q string) ([]FacetValueCount, bool, error) {
	if err := validateMetadataRequest(req); err != nil {
		return nil, false, err
	}
	if strings.TrimSpace(dataTable) == "" {
		dataTable = "public.data_items"
	}
	if strings.TrimSpace(schemaTable) == "" {
		schemaTable = postgresResponseSchemaTable
	}
	ds, rows, _, err := loadPostgresDataSource(req, dsn, dataTable, schemaTable)
	if err != nil {
		return nil, false, err
	}
	return facetValuesForRows(ds, rows, field, limit, q)
}

func validateMetadataRequest(req FilterRequest) error {
	if err := validateRequestSchemaVersion(req.SchemaVersion); err != nil {
		return err
	}
	if !isUUID(req.RequestUUID) {
		return fmt.Errorf("invalid RequestUuid: %q", req.RequestUUID)
	}
	if !isUUID(req.DataSourceUUID) {
		return fmt.Errorf("invalid DataSourceUuid: %q", req.DataSourceUUID)
	}
	if strings.TrimSpace(req.DataSourceName) == "" {
		return fmt.Errorf("DataSourceName is required")
	}
	return nil
}

func facetValuesForRows(ds DataSourceDefinition, rows []map[string]interface{}, field string, limit int, q string) ([]FacetValueCount, bool, error) {
	if strings.TrimSpace(field) == "" {
		return nil, false, fmt.Errorf("field is required")
	}
	if _, ok := ds.Fields[field]; !ok {
		return nil, false, fmt.Errorf("unknown field: %q", field)
	}

	needle := strings.ToLower(strings.TrimSpace(q))
	counts := make(map[string]int)
	values := make(map[string]FacetValueCount)

	for _, row := range rows {
		val, ok := row[field]
		if !ok {
			val = nil
		}
		key := facetKey(val)
		label := facetLabel(val)
		if needle != "" && !strings.Contains(strings.ToLower(label), needle) {
			continue
		}
		counts[key]++
		values[key] = FacetValueCount{
			Value:  val,
			IsNull: val == nil,
		}
	}

	out := make([]FacetValueCount, 0, len(values))
	for key, item := range values {
		item.Count = counts[key]
		out = append(out, item)
	}

	sort.Slice(out, func(i, j int) bool {
		if out[i].Count != out[j].Count {
			return out[i].Count > out[j].Count
		}
		return facetLabel(out[i].Value) < facetLabel(out[j].Value)
	})

	truncated := false
	if limit > 0 && len(out) > limit {
		out = out[:limit]
		truncated = true
	}

	return out, truncated, nil
}

func facetKey(v interface{}) string {
	if v == nil {
		return "null"
	}
	return fmt.Sprintf("%T:%v", v, v)
}

func facetLabel(v interface{}) string {
	if v == nil {
		return "(blank)"
	}
	return fmt.Sprintf("%v", v)
}

func describeDataSourceFromSQLiteSchema(req FilterRequest, dbPath string) (AllowedFieldResponse, error) {
	if strings.TrimSpace(dbPath) == "" {
		return AllowedFieldResponse{}, fmt.Errorf("db path is required")
	}
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return AllowedFieldResponse{}, fmt.Errorf("open sqlite db: %w", err)
	}
	defer db.Close()

	schemaMeta, err := loadDataSetSchemaMetadata(context.Background(), db, req)
	if err != nil {
		return AllowedFieldResponse{}, err
	}
	return allowedFieldsFromSchema(req, schemaMeta)
}

func describeDataSourceFromPostgresSchema(req FilterRequest, dsn string, schemaTable string) (AllowedFieldResponse, error) {
	if strings.TrimSpace(dsn) == "" {
		return AllowedFieldResponse{}, fmt.Errorf("postgres dsn is required")
	}
	if strings.TrimSpace(schemaTable) == "" {
		schemaTable = postgresResponseSchemaTable
	}
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return AllowedFieldResponse{}, fmt.Errorf("open postgres db: %w", err)
	}
	defer db.Close()

	schemaMeta, err := loadPostgresDataSetSchemaMetadata(context.Background(), db, req, schemaTable)
	if err != nil {
		return AllowedFieldResponse{}, err
	}
	return allowedFieldsFromSchema(req, schemaMeta)
}

func allowedFieldsFromSchema(req FilterRequest, schemaMeta *DataSetSchemaMetadata) (AllowedFieldResponse, error) {
	catalog, err := schemaCatalogForDataSource(schemaMeta)
	if err != nil {
		return AllowedFieldResponse{}, err
	}

	ds := schemaDataSourceDefinition(catalog, req.DataSourceUUID)
	return allowedFieldsForDataSource(req, ds)
}
