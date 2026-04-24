package filters

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// TestQueryCSVDataSource validates end-to-end request filtering against CSV input.
func TestQueryCSVDataSource(t *testing.T) {
	t.Parallel()

	input := []byte(`{
	  "SchemaVersion": "1.0",
	  "RequestUuid": "6e6e17c4-6cc0-4ef0-a1cf-e96f0c5f8b8f",
	  "DataSourceUuid": "110cc994-a913-4041-96fe-a96d7e0c97e8",
	  "DataSourceName": "SubCustody",
	  "RequestFilter": {
	    "and": [
	      {
	        "field": "AccountCurrency",
	        "op": "eq",
	        "value": "SEK"
	      },
	      {
	        "field": "AccountEnvironment",
	        "op": "eq",
	        "value": "SysTest"
	      },
	      {
	        "field": "ClientJurisdictionCountryCode",
	        "op": "eq",
	        "value": "SE"
	      }
	    ]
	  }
	}`)

	var req FilterRequest
	if err := json.Unmarshal(input, &req); err != nil {
		t.Fatalf("unmarshal request: %v", err)
	}
	logUnitCall(t, "11111111-1111-4111-8111-111111111111", "json.Unmarshal(FilterRequest)", string(input), "valid FilterRequest", req)

	csvPath := filepath.Join("..", "..", "P26_2", "FenixRawTestdata_646rows_211220_stripped.csv")
	compiled, allowed, dataResp, err := QueryCSVDataSource(req, csvPath, 1)
	logUnitCall(t, "11111111-1111-4111-8111-111111111111", "QueryCSVDataSource", map[string]any{"request": req, "csvPath": csvPath, "maxItems": 1}, "compiled+allowed+1 row", map[string]any{"compiled": compiled, "allowedCount": len(allowed.AllowedFields), "rows": len(dataResp.Data), "err": err})
	if err != nil {
		t.Fatalf("QueryCSVDataSource failed: %v", err)
	}

	if compiled.WhereSQL == "" {
		t.Fatal("expected non-empty compiled SQL")
	}
	if len(allowed.AllowedFields) == 0 {
		t.Fatal("expected inferred allowed fields")
	}
	if len(dataResp.Data) != 1 {
		t.Fatalf("expected one matching row, got %d", len(dataResp.Data))
	}
	if got := dataResp.Data[0]["AccountCurrency"]; got != "SEK" {
		t.Fatalf("expected AccountCurrency SEK, got %#v", got)
	}
	if got := dataResp.Data[0]["ClientJurisdictionCountryCode"]; got != "SE" {
		t.Fatalf("expected canonical ClientJurisdictionCountryCode SE, got %#v", got)
	}
}

func TestQueryCSVDataSourceUsesCanonicalSchemaFields(t *testing.T) {
	t.Parallel()

	csvPath := filepath.Join(t.TempDir(), "source.csv")
	payload := "AccountCurrency;ClientJuristictionCountryCode\nSEK;SE\nNOK;NO\n"
	if err := os.WriteFile(csvPath, []byte(payload), 0o644); err != nil {
		t.Fatalf("write csv fixture: %v", err)
	}

	req := FilterRequest{
		SchemaVersion:  "1.0",
		RequestUUID:    "7e7e17c4-6cc0-4ef0-a1cf-e96f0c5f8b8f",
		DataSourceUUID: "110cc994-a913-4041-96fe-a96d7e0c97e8",
		DataSourceName: "SubCustody",
		RequestFilter:  []byte(`{"field":"ClientJurisdictionCountryCode","op":"eq","value":"SE"}`),
	}

	_, allowed, dataResp, err := QueryCSVDataSource(req, csvPath, 0)
	if err != nil {
		t.Fatalf("QueryCSVDataSource unexpected error: %v", err)
	}

	foundCanonicalField := false
	for _, field := range allowed.AllowedFields {
		if field.FieldName == "ClientJurisdictionCountryCode" {
			foundCanonicalField = true
			break
		}
	}
	if !foundCanonicalField {
		t.Fatal("expected canonical schema field ClientJurisdictionCountryCode to be allowed")
	}

	if len(dataResp.Data) != 1 {
		t.Fatalf("expected one matching row for canonical country code filter, got %d", len(dataResp.Data))
	}
	if got := dataResp.Data[0]["ClientJurisdictionCountryCode"]; got != "SE" {
		t.Fatalf("expected canonical ClientJurisdictionCountryCode=SE, got %#v", got)
	}
	if _, hasRaw := dataResp.Data[0]["ClientJuristictionCountryCode"]; hasRaw {
		t.Fatal("unexpected raw legacy field name in csv response row")
	}
}
