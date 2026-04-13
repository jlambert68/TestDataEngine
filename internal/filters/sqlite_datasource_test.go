package filters

import (
	"context"
	"database/sql"
	"encoding/json"
	"path/filepath"
	"reflect"
	"testing"

	_ "modernc.org/sqlite"
)

// TestQuerySQLiteDataSource validates end-to-end filtering with SQLite as source.
func TestQuerySQLiteDataSource(t *testing.T) {
	t.Parallel()

	dbPath := filepath.Join(t.TempDir(), "source.sqlite")
	createSQLiteTestData(t, dbPath)

	req := FilterRequest{
		SchemaVersion:  "1.0",
		RequestUUID:    "6e6e17c4-6cc0-4ef0-a1cf-e96f0c5f8b8f",
		DataSourceUUID: "110cc994-a913-4041-96fe-a96d7e0c97e8",
		DataSourceName: "SubCustody",
		RequestFilter:  []byte(`{"field":"AccountCurrency","op":"eq","value":"SEK"}`),
	}

	compiled, allowed, dataResp, err := QuerySQLiteDataSource(req, dbPath, "main.data_items", 1)
	logUnitCall(t, "11111111-1111-4111-8111-111111111111", "QuerySQLiteDataSource", map[string]any{"dbPath": dbPath, "maxItems": 1}, "compiled+allowed+1 row", map[string]any{"compiled": compiled, "allowedCount": len(allowed.AllowedFields), "rows": len(dataResp.Data), "err": err})
	if err != nil {
		t.Fatalf("QuerySQLiteDataSource unexpected error: %v", err)
	}
	if compiled.WhereSQL == "" {
		t.Fatal("expected compiled SQL")
	}
	if len(allowed.AllowedFields) == 0 {
		t.Fatal("expected inferred allowed fields")
	}
	if len(dataResp.Data) != 1 {
		t.Fatalf("expected one row because maxItems=1, got %d", len(dataResp.Data))
	}
	if got := dataResp.Data[0]["AccountCurrency"]; got != "SEK" {
		t.Fatalf("expected AccountCurrency=SEK, got %#v", got)
	}
	if dataResp.TestDataSourceName != req.DataSourceName {
		t.Fatalf("expected TestDataSourceName=%q, got %q", req.DataSourceName, dataResp.TestDataSourceName)
	}
	if dataResp.TestDataSourceUUID != req.DataSourceUUID {
		t.Fatalf("expected TestDataSourceUuid=%q, got %q", req.DataSourceUUID, dataResp.TestDataSourceUUID)
	}
	if dataResp.JsonSchemaName != SpecificDatasourceResponseSchemaName {
		t.Fatalf("unexpected JsonSchemaName: %q", dataResp.JsonSchemaName)
	}
	if !json.Valid(dataResp.JsonSchema) {
		t.Fatalf("expected JsonSchema to contain valid JSON, got %q", string(dataResp.JsonSchema))
	}
	if dataResp.UpdatedDateTime == "" {
		t.Fatal("expected UpdatedDateTime to be populated from schema metadata table")
	}

	_, _, unboundedResp, err := QuerySQLiteDataSource(req, dbPath, "main.data_items", 0)
	logUnitCall(t, "11111111-1111-4111-8111-111111111111", "QuerySQLiteDataSource", map[string]any{"dbPath": dbPath, "maxItems": 0}, "all matching rows", map[string]any{"rows": len(unboundedResp.Data), "err": err})
	if err != nil {
		t.Fatalf("QuerySQLiteDataSource(maxItems=0) unexpected error: %v", err)
	}
	if len(unboundedResp.Data) != 2 {
		t.Fatalf("expected 2 matching rows, got %d", len(unboundedResp.Data))
	}

	seedGUID := "cccccccc-cccc-4ccc-8ccc-cccccccccccc"
	_, _, seededFirst, err := QuerySQLiteDataSourceWithSeed(req, dbPath, "main.data_items", 2, seedGUID)
	logUnitCall(t, "11111111-1111-4111-8111-111111111111", "QuerySQLiteDataSourceWithSeed", map[string]any{"seed": seedGUID, "run": "first"}, "nil error + deterministic rows", map[string]any{"rows": seededFirst.Data, "err": err})
	if err != nil {
		t.Fatalf("QuerySQLiteDataSourceWithSeed first call unexpected error: %v", err)
	}
	_, _, seededSecond, err := QuerySQLiteDataSourceWithSeed(req, dbPath, "main.data_items", 2, seedGUID)
	logUnitCall(t, "11111111-1111-4111-8111-111111111111", "QuerySQLiteDataSourceWithSeed", map[string]any{"seed": seedGUID, "run": "second"}, "nil error + deterministic rows", map[string]any{"rows": seededSecond.Data, "err": err})
	if err != nil {
		t.Fatalf("QuerySQLiteDataSourceWithSeed second call unexpected error: %v", err)
	}
	if !reflect.DeepEqual(seededFirst.Data, seededSecond.Data) {
		t.Fatal("expected deterministic rows for same sqlite seed guid")
	}
}

// TestQuerySQLiteDataSourceErrors validates guard rails for invalid SQLite inputs.
func TestQuerySQLiteDataSourceErrors(t *testing.T) {
	t.Parallel()

	req := FilterRequest{
		SchemaVersion:  "1.0",
		RequestUUID:    "6e6e17c4-6cc0-4ef0-a1cf-e96f0c5f8b8f",
		DataSourceUUID: "110cc994-a913-4041-96fe-a96d7e0c97e8",
		DataSourceName: "SubCustody",
		RequestFilter:  []byte(`{"field":"AccountCurrency","op":"eq","value":"SEK"}`),
	}

	_, _, _, err := QuerySQLiteDataSource(req, "", "main.data_items", 1)
	logUnitCall(t, "11111111-1111-4111-8111-111111111111", "QuerySQLiteDataSource", map[string]any{"dbPath": ""}, "error", err)
	if err == nil {
		t.Fatal("expected db path validation error")
	}

	_, _, _, err = QuerySQLiteDataSource(req, "/tmp/x.sqlite", "main.data_items;drop table x", 1)
	logUnitCall(t, "11111111-1111-4111-8111-111111111111", "QuerySQLiteDataSource", map[string]any{"tableName": "main.data_items;drop table x"}, "error", err)
	if err == nil {
		t.Fatal("expected unsafe table name error")
	}
}

// TestSQLiteHelpers covers small helper functions used by the SQLite adapter.
func TestSQLiteHelpers(t *testing.T) {
	if !isSafeTableIdentifier("main.data_items") {
		t.Fatal("expected safe table identifier")
	}
	if isSafeTableIdentifier("main.data_items;drop") {
		t.Fatal("expected unsafe table identifier to be rejected")
	}

	fields := collectFieldOrder([]map[string]interface{}{
		{"b": "1", "a": "2"},
		{"c": "3"},
	})
	if len(fields) != 3 || fields[0] != "a" || fields[1] != "b" || fields[2] != "c" {
		t.Fatalf("unexpected field order: %#v", fields)
	}

	if v := rawValueToStringForInference(true); v != "true" {
		t.Fatalf("unexpected bool conversion: %q", v)
	}
	if v := rawValueToStringForInference(10.5); v != "10.5" {
		t.Fatalf("unexpected float conversion: %q", v)
	}
	if v := rawValueToStringForInference(nil); v != "NULL" {
		t.Fatalf("unexpected nil conversion: %q", v)
	}

	got, err := coerceRawValue("12", "integer")
	if err != nil {
		t.Fatalf("coerceRawValue unexpected error: %v", err)
	}
	if got.(int64) != 12 {
		t.Fatalf("unexpected coerced integer: %#v", got)
	}
}

// createSQLiteTestData builds a temporary SQLite dataset for integration-style tests.
func createSQLiteTestData(t *testing.T, dbPath string) {
	t.Helper()

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	defer db.Close()

	schema := `
create table main.data_items
(
    DataSourceUuid      TEXT not null,
    DataSourceName      TEXT not null,
    DataUuid            TEXT not null,
    DataUpdateTimeStamp TEXT not null,
    JsonDataUuid        TEXT not null primary key,
    JsonData            TEXT not null,
    check (json_valid(JsonData))
);`
	if _, err := db.ExecContext(context.Background(), schema); err != nil {
		t.Fatalf("create schema: %v", err)
	}

	metadataSchema := `
create table main.testdataset_response_schemas
(
    TestDataSourceName TEXT not null,
    TestDataSourceUuid TEXT not null,
    JsonSchemaName     TEXT not null,
    JsonSchema         TEXT not null,
    UpdatedDateTime    TEXT not null
);`
	if _, err := db.ExecContext(context.Background(), metadataSchema); err != nil {
		t.Fatalf("create metadata schema: %v", err)
	}

	insert := `
insert into main.data_items
  (DataSourceUuid, DataSourceName, DataUuid, DataUpdateTimeStamp, JsonDataUuid, JsonData)
values (?, ?, ?, ?, ?, ?)`

	rows := []struct {
		dataUUID     string
		jsonDataUUID string
		jsonData     string
	}{
		{
			dataUUID:     "11111111-1111-4111-8111-111111111111",
			jsonDataUUID: "21111111-1111-4111-8111-111111111111",
			jsonData:     `{"AccountCurrency":"SEK","Amount":"100","Flag":"true"}`,
		},
		{
			dataUUID:     "11111111-1111-4111-8111-111111111112",
			jsonDataUUID: "21111111-1111-4111-8111-111111111112",
			jsonData:     `{"AccountCurrency":"SEK","Amount":"200","Flag":"false"}`,
		},
		{
			dataUUID:     "11111111-1111-4111-8111-111111111113",
			jsonDataUUID: "21111111-1111-4111-8111-111111111113",
			jsonData:     `{"AccountCurrency":"NOK","Amount":"300","Flag":"true"}`,
		},
	}
	for _, row := range rows {
		if _, err := db.ExecContext(
			context.Background(),
			insert,
			"110cc994-a913-4041-96fe-a96d7e0c97e8",
			"SubCustody",
			row.dataUUID,
			"2026-03-26T10:00:00Z",
			row.jsonDataUUID,
			row.jsonData,
		); err != nil {
			t.Fatalf("insert test row: %v", err)
		}
	}

	const responseSchemaJSON = `{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "title": "TestDataSet response for a specific datasource",
  "type": "object",
  "additionalProperties": false,
  "required": ["SchemaVersion", "TestDataSourceName", "TestDataSourceUuid", "JsonSchemaName", "TestData"],
  "properties": {
    "SchemaVersion": {
      "type": "string",
      "enum": ["1.0"]
    },
    "TestDataSourceName": {
      "type": "string"
    },
    "TestDataSourceUuid": {
      "type": "string",
      "format": "uuid"
    },
    "JsonSchemaName": {
      "type": "string"
    },
    "TestData": {
      "$ref": "#/$defs/TestData"
    }
  },
  "$defs": {
    "TestData": {
      "type": "object",
      "additionalProperties": false,
      "required": ["SpecificSourceSchemaVersion", "TestDataSet"],
      "properties": {
        "SpecificSourceSchemaVersion": {
          "type": "string",
          "enum": ["1.0"]
        },
        "TestDataSet": {
          "type": "array",
          "items": {
            "$ref": "#/$defs/TestDataSetItem"
          }
        }
      }
    },
    "TestDataSetItem": {
      "type": "object",
      "additionalProperties": false
    }
  }
}`
	if _, err := db.ExecContext(
		context.Background(),
		`insert into main.testdataset_response_schemas
  (TestDataSourceName, TestDataSourceUuid, JsonSchemaName, JsonSchema, UpdatedDateTime)
values (?, ?, ?, ?, ?)`,
		"SubCustody",
		"110cc994-a913-4041-96fe-a96d7e0c97e8",
		SpecificDatasourceResponseSchemaName,
		responseSchemaJSON,
		"2026-04-09T10:00:00Z",
	); err != nil {
		t.Fatalf("insert metadata row: %v", err)
	}
}
