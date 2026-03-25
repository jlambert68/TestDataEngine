package filters

import (
	"encoding/json"
	"path/filepath"
	"testing"
)

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
	        "field": "ClientJuristictionCountryCode",
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
}
