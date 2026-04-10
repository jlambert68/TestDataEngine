package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"TestDataEngine/internal/filters"
)

func TestValidateSQLiteRequestSchema(t *testing.T) {
	t.Parallel()

	valid := []byte(`{
	  "SchemaVersion": "1.0",
	  "RequestUuid": "6e6e17c4-6cc0-4ef0-a1cf-e96f0c5f8b8f",
	  "DataSourceUuid": "110cc994-a913-4041-96fe-a96d7e0c97e8",
	  "DataSourceName": "SubCustody",
	  "RequestFilter": {
	    "field": "AccountCurrency",
	    "op": "eq",
	    "value": "SEK"
	  }
	}`)
	if err := validateSQLiteRequestSchema(valid); err != nil {
		t.Fatalf("expected valid sqlite request schema, got error: %v", err)
	}

	invalid := []byte(`{
	  "SchemaVersion": "1.0",
	  "RequestUuid": "6e6e17c4-6cc0-4ef0-a1cf-e96f0c5f8b8f",
	  "DataSourceUuid": "110cc994-a913-4041-96fe-a96d7e0c97e8",
	  "DataSourceName": "SubCustody"
	}`)
	if err := validateSQLiteRequestSchema(invalid); err == nil {
		t.Fatal("expected schema validation error for missing RequestFilter")
	}
}

func TestValidateSQLiteResponseSchema(t *testing.T) {
	t.Parallel()

	resp := filters.DataSetResponse{
		SchemaVersion:      "1.0",
		TestDataSourceName: "SubCustody",
		TestDataSourceUUID: "110cc994-a913-4041-96fe-a96d7e0c97e8",
		JsonSchemaName:     filters.SpecificDatasourceResponseSchemaName,
		JsonSchema:         json.RawMessage(`{"type":"object"}`),
		UpdatedDateTime:    "2026-04-09T00:00:00Z",
		DataSourceName:     "SubCustody",
		DataSourceUUID:     "110cc994-a913-4041-96fe-a96d7e0c97e8",
		Data:               []map[string]interface{}{},
	}
	if err := validateSQLiteResponseSchema(resp); err != nil {
		t.Fatalf("expected valid sqlite response schema, got error: %v", err)
	}

	resp.JsonSchemaName = ""
	if err := validateSQLiteResponseSchema(resp); err == nil {
		t.Fatal("expected schema validation error for empty JsonSchemaName")
	}
}

func TestValidateSQLiteResponseSchemaLegacySchemaNameAlias(t *testing.T) {
	t.Parallel()

	resp := filters.DataSetResponse{
		SchemaVersion:      "1.0",
		TestDataSourceName: "SubCustody",
		TestDataSourceUUID: "110cc994-a913-4041-96fe-a96d7e0c97e8",
		JsonSchemaName:     "TestDataSet_Response_For_Specific_DatasourceFrom_TestDataEngine.json-schema.json",
		DataSourceName:     "SubCustody",
		DataSourceUUID:     "110cc994-a913-4041-96fe-a96d7e0c97e8",
		Data:               []map[string]interface{}{},
	}
	if err := validateSQLiteResponseSchema(resp); err != nil {
		t.Fatalf("expected legacy sqlite schema filename alias to validate, got error: %v", err)
	}
}

func TestValidateCSVResponseSchema(t *testing.T) {
	t.Parallel()

	resp := filters.DataSetResponse{
		SchemaVersion:      "1.0",
		TestDataSourceName: "SubCustody",
		TestDataSourceUUID: "110cc994-a913-4041-96fe-a96d7e0c97e8",
		JsonSchemaName:     filters.SpecificDatasourceResponseSchemaName,
		JsonSchema:         json.RawMessage(`{"type":"object"}`),
		UpdatedDateTime:    "2026-04-09T00:00:00Z",
		DataSourceName:     "SubCustody",
		DataSourceUUID:     "110cc994-a913-4041-96fe-a96d7e0c97e8",
		Data:               []map[string]interface{}{},
	}
	if err := validateCSVResponseSchema(resp); err != nil {
		t.Fatalf("expected valid csv response schema, got error: %v", err)
	}
}

func TestValidateSQLiteResponseSchemaWithPathTraversalSchemaName(t *testing.T) {
	t.Parallel()

	resp := filters.DataSetResponse{
		SchemaVersion:      "1.0",
		TestDataSourceName: "SubCustody",
		TestDataSourceUUID: "110cc994-a913-4041-96fe-a96d7e0c97e8",
		JsonSchemaName:     "../../TestDataSet_Response_For_Specific_Datasource_From_TestDataEngine.json-schema.json",
		JsonSchema:         json.RawMessage(`{"type":"object"}`),
		UpdatedDateTime:    "2026-04-09T00:00:00Z",
		DataSourceName:     "SubCustody",
		DataSourceUUID:     "110cc994-a913-4041-96fe-a96d7e0c97e8",
		Data:               []map[string]interface{}{},
	}
	if err := validateSQLiteResponseSchema(resp); err != nil {
		t.Fatalf("expected path-traversal-style JsonSchemaName to still resolve by basename, got: %v", err)
	}
}

func TestValidateSQLiteResponseSchemaUnknownSchemaName(t *testing.T) {
	t.Parallel()

	resp := filters.DataSetResponse{
		SchemaVersion:      "1.0",
		TestDataSourceName: "SubCustody",
		TestDataSourceUUID: "110cc994-a913-4041-96fe-a96d7e0c97e8",
		JsonSchemaName:     "does-not-exist.json",
		JsonSchema:         json.RawMessage(`{"type":"object"}`),
		UpdatedDateTime:    "2026-04-09T00:00:00Z",
		DataSourceName:     "SubCustody",
		DataSourceUUID:     "110cc994-a913-4041-96fe-a96d7e0c97e8",
		Data:               []map[string]interface{}{},
	}
	if err := validateSQLiteResponseSchema(resp); err == nil {
		t.Fatal("expected validation error for unknown JsonSchemaName")
	}
}

func TestValidateJSONAgainstSchemaFileErrors(t *testing.T) {
	t.Parallel()

	tempSchemaPath := filepath.Join(t.TempDir(), "schema.json")
	schemaJSON := []byte(`{"$schema":"https://json-schema.org/draft/2020-12/schema","type":"object","required":["x"]}`)
	if err := os.WriteFile(tempSchemaPath, schemaJSON, 0o600); err != nil {
		t.Fatalf("write temp schema: %v", err)
	}

	if err := validateJSONAgainstSchemaFile(tempSchemaPath, []byte(`{"x":1}`)); err != nil {
		t.Fatalf("expected valid payload against temp schema, got: %v", err)
	}

	if err := validateJSONAgainstSchemaFile(tempSchemaPath, []byte(`{`)); err == nil {
		t.Fatal("expected decode payload JSON error for invalid JSON payload")
	}

	err := validateJSONAgainstSchemaFile("missing-schema-file.json", []byte(`{"x":1}`))
	if err == nil {
		t.Fatal("expected resolve schema path error for missing schema file")
	}
	if !strings.Contains(err.Error(), "no matching file found") {
		t.Fatalf("expected missing-file error text, got: %v", err)
	}
}

func TestApplyLocalResponseSchemaMetadata(t *testing.T) {
	t.Parallel()

	req := filters.FilterRequest{
		SchemaVersion:  "1.0",
		RequestUUID:    "6e6e17c4-6cc0-4ef0-a1cf-e96f0c5f8b8f",
		DataSourceUUID: "110cc994-a913-4041-96fe-a96d7e0c97e8",
		DataSourceName: "SubCustody",
		RequestFilter:  json.RawMessage(`{"field":"AccountCurrency","op":"eq","value":"SEK"}`),
	}
	resp := filters.DataSetResponse{
		DataSourceName: req.DataSourceName,
		DataSourceUUID: req.DataSourceUUID,
		Data:           []map[string]interface{}{{"AccountCurrency": "SEK"}},
	}

	enriched, err := applyLocalResponseSchemaMetadata(req, resp, filters.SpecificDatasourceResponseSchemaName)
	if err != nil {
		t.Fatalf("expected metadata enrichment to succeed, got error: %v", err)
	}
	if enriched.TestDataSourceName != req.DataSourceName {
		t.Fatalf("unexpected TestDataSourceName: %q", enriched.TestDataSourceName)
	}
	if enriched.TestDataSourceUUID != req.DataSourceUUID {
		t.Fatalf("unexpected TestDataSourceUuid: %q", enriched.TestDataSourceUUID)
	}
	if enriched.JsonSchemaName == "" || len(enriched.JsonSchema) == 0 || enriched.UpdatedDateTime == "" {
		t.Fatalf("expected schema metadata fields to be populated: %#v", enriched)
	}
	if !json.Valid(enriched.JsonSchema) {
		t.Fatalf("expected JsonSchema to be valid JSON, got %q", string(enriched.JsonSchema))
	}
}

func TestApplyLocalResponseSchemaMetadataErrorsAndBasename(t *testing.T) {
	t.Parallel()

	req := filters.FilterRequest{
		SchemaVersion:  "1.0",
		RequestUUID:    "6e6e17c4-6cc0-4ef0-a1cf-e96f0c5f8b8f",
		DataSourceUUID: "110cc994-a913-4041-96fe-a96d7e0c97e8",
		DataSourceName: "SubCustody",
		RequestFilter:  json.RawMessage(`{"field":"AccountCurrency","op":"eq","value":"SEK"}`),
	}
	resp := filters.DataSetResponse{
		DataSourceName: req.DataSourceName,
		DataSourceUUID: req.DataSourceUUID,
		Data:           []map[string]interface{}{{"AccountCurrency": "SEK"}},
	}

	if _, err := applyLocalResponseSchemaMetadata(req, resp, "does-not-exist.json"); err == nil {
		t.Fatal("expected schema metadata enrichment error for missing schema file")
	}

	enriched, err := applyLocalResponseSchemaMetadata(
		req,
		resp,
		"../../TestDataSet_Response_For_Specific_Datasource_From_TestDataEngine.json-schema.json",
	)
	if err != nil {
		t.Fatalf("expected basename schema name resolution to succeed, got: %v", err)
	}
	if enriched.JsonSchemaName != filters.SpecificDatasourceResponseSchemaName {
		t.Fatalf("expected basename-only JsonSchemaName, got %q", enriched.JsonSchemaName)
	}
}
