package main

import (
	"encoding/json"
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
		TestDataSourceName: "SubCustody",
		TestDataSourceUUID: "110cc994-a913-4041-96fe-a96d7e0c97e8",
		JsonSchemaName:     "TestDataSet_Response_For_Specific_DatasourceFrom_TestDataEngine.json-schema.json",
		JsonSchema:         json.RawMessage(`{"type":"object"}`),
		UpdatedDateTime:    "2026-04-09T00:00:00Z",
		DataSourceName:     "SubCustody",
		DataSourceUUID:     "110cc994-a913-4041-96fe-a96d7e0c97e8",
		Data: []map[string]interface{}{
			{"AccountCurrency": "SEK"},
		},
	}
	if err := validateSQLiteResponseSchema(resp); err != nil {
		t.Fatalf("expected valid sqlite response schema, got error: %v", err)
	}

	resp.JsonSchemaName = ""
	if err := validateSQLiteResponseSchema(resp); err == nil {
		t.Fatal("expected schema validation error for empty JsonSchemaName")
	}
}

func TestValidateCSVResponseSchema(t *testing.T) {
	t.Parallel()

	resp := filters.DataSetResponse{
		TestDataSourceName: "SubCustody",
		TestDataSourceUUID: "110cc994-a913-4041-96fe-a96d7e0c97e8",
		JsonSchemaName:     "TestDataSet_Response_For_Specific_DatasourceFrom_TestDataEngine.json-schema.json",
		JsonSchema:         json.RawMessage(`{"type":"object"}`),
		UpdatedDateTime:    "2026-04-09T00:00:00Z",
		DataSourceName:     "SubCustody",
		DataSourceUUID:     "110cc994-a913-4041-96fe-a96d7e0c97e8",
		Data: []map[string]interface{}{
			{"AccountCurrency": "SEK"},
		},
	}
	if err := validateCSVResponseSchema(resp); err != nil {
		t.Fatalf("expected valid csv response schema, got error: %v", err)
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

	enriched, err := applyLocalResponseSchemaMetadata(req, resp, "TestDataSet_Response_For_Specific_DatasourceFrom_TestDataEngine.json-schema.json")
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
