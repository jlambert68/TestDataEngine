package main

import (
	"encoding/json"
	"flag"
	"os"
	"strings"

	"TestDataEngine/internal/filters"
	"TestDataEngine/internal/logging"
)

// main runs a sample filter request and prints both metadata and matching data rows.
func main() {
	// Embedded example request used by default when running the binary directly.
	filterReqJSON := []byte(`{
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

	var req filters.FilterRequest
	if err := json.Unmarshal(filterReqJSON, &req); err != nil {
		logging.Fatalf("7e7e323b-a072-4af3-8608-95316cce6fe2", "failed to unmarshal filter request: %v", err)
	}

	var (
		sourceType  string
		csvPath     string
		sqliteDB    string
		sqliteTable string
		maxItems    int
		randomGUID  string
	)
	flag.StringVar(&sourceType, "source", "csv", "Data source type: csv or sqlite")
	flag.StringVar(&csvPath, "csv", "p26_2/FenixRawTestdata_646rows_211220_stripped.csv", "Path to CSV input file")
	flag.StringVar(&sqliteDB, "sqlite-db", "testdata/SQLiteDB/identifier.sqlite", "Path to SQLite DB file")
	flag.StringVar(&sqliteTable, "sqlite-table", "main.data_items", "SQLite table containing JsonData")
	flag.IntVar(&maxItems, "max-items", 2, "Maximum number of matched rows to return (0=all)")
	flag.StringVar(&randomGUID, "random-seed-guid", "", "Optional GUID used as deterministic shuffle seed")
	flag.Parse()

	// Negative values are treated as unbounded to avoid surprising hard failures.
	if maxItems < 0 {
		maxItems = 0
	}

	var (
		compiled    filters.CompiledFilter
		allowedResp filters.AllowedFieldResponse
		dataResp    filters.DataSetResponse
		err         error
	)

	switch strings.ToLower(strings.TrimSpace(sourceType)) {
	case "csv":
		// Keep backward compatibility with both lowercase and uppercase data folders.
		if _, statErr := os.Stat(csvPath); statErr != nil && csvPath == "p26_2/FenixRawTestdata_646rows_211220_stripped.csv" {
			csvPath = "P26_2/FenixRawTestdata_646rows_211220_stripped.csv"
		}
		compiled, allowedResp, dataResp, err = filters.QueryCSVDataSourceWithSeed(req, csvPath, maxItems, randomGUID)
		if err != nil {
			logging.Fatalf("c2fd3f4f-1119-47d8-bbe7-28f159f57db2", "failed to query csv datasource: %v", err)
		}
		logging.Infof("3fd182f4-3d81-4225-b89f-f2dc959fc8ba", "Source=csv CSV=%s RandomSeedGuid=%s", csvPath, randomGUID)

	case "sqlite":
		compiled, allowedResp, dataResp, err = filters.QuerySQLiteDataSourceWithSeed(req, sqliteDB, sqliteTable, maxItems, randomGUID)
		if err != nil {
			logging.Fatalf("c2fd3f4f-1119-47d8-bbe7-28f159f57db2", "failed to query sqlite datasource: %v", err)
		}
		logging.Infof("3fd182f4-3d81-4225-b89f-f2dc959fc8ba", "Source=sqlite DB=%s Table=%s RandomSeedGuid=%s", sqliteDB, sqliteTable, randomGUID)

	default:
		logging.Fatalf("a72f852f-bf0a-40de-bbe7-b54225095f20", "unsupported source type %q (expected csv or sqlite)", sourceType)
	}

	// Log compiled SQL representation and response payloads for traceability.
	logging.Infof("35579f2f-4de2-4cc2-bf0a-bf579f31cf64", "WHERE=%s", compiled.WhereSQL)
	logging.Infof("37f52f2f-fb8f-47dd-bf14-17c3a194ddbc", "ARGS=%v", compiled.Args)

	allowedWithInputFilter := struct {
		InputFilter           json.RawMessage              `json:"InputFilter"`
		AllowedFieldsResponse filters.AllowedFieldResponse `json:"AllowedFieldsResponse"`
	}{
		InputFilter:           req.RequestFilter,
		AllowedFieldsResponse: allowedResp,
	}
	allowedPretty, err := json.MarshalIndent(allowedWithInputFilter, "", "  ")
	if err != nil {
		logging.Fatalf("2e8c5ee6-241d-4f65-b82a-0877eef3644d", "failed to marshal allowed fields response: %v", err)
	}
	logging.Infof("15f177af-c4dc-4e86-a4e5-4f20fdf001d3", "AllowedFieldsResponse=%s", string(allowedPretty))

	dataWithInputFilter := struct {
		InputFilter     json.RawMessage         `json:"InputFilter"`
		DataSetResponse filters.DataSetResponse `json:"DataSetResponse"`
	}{
		InputFilter:     req.RequestFilter,
		DataSetResponse: dataResp,
	}
	dataPretty, err := json.MarshalIndent(dataWithInputFilter, "", "  ")
	if err != nil {
		logging.Fatalf("9efef5d2-f500-450f-929f-890f4d89f777", "failed to marshal data response: %v", err)
	}
	logging.Infof("70e0f6f2-72fd-42bf-9f0e-f9afca6ebc52", "DataSetResponse=%s", string(dataPretty))
}
