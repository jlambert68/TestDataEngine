package main

import (
	"encoding/json"
	"os"

	"TestDataEngine/internal/filters"
	"TestDataEngine/internal/logging"
)

func main() {
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

	csvDataSourcePath := "p26_2/FenixRawTestdata_646rows_211220_stripped.csv"
	if _, err := os.Stat(csvDataSourcePath); err != nil {
		csvDataSourcePath = "P26_2/FenixRawTestdata_646rows_211220_stripped.csv"
	}

	compiled, allowedResp, dataResp, err := filters.QueryCSVDataSource(req, csvDataSourcePath, 2)
	if err != nil {
		logging.Fatalf("c2fd3f4f-1119-47d8-bbe7-28f159f57db2", "failed to query csv datasource: %v", err)
	}

	logging.Infof("3fd182f4-3d81-4225-b89f-f2dc959fc8ba", "CSV=%s", csvDataSourcePath)
	logging.Infof("35579f2f-4de2-4cc2-bf0a-bf579f31cf64", "WHERE=%s", compiled.WhereSQL)
	logging.Infof("37f52f2f-fb8f-47dd-bf14-17c3a194ddbc", "ARGS=%v", compiled.Args)

	allowedPretty, err := json.MarshalIndent(allowedResp, "", "  ")
	if err != nil {
		logging.Fatalf("2e8c5ee6-241d-4f65-b82a-0877eef3644d", "failed to marshal allowed fields response: %v", err)
	}
	logging.Infof("15f177af-c4dc-4e86-a4e5-4f20fdf001d3", "AllowedFieldsResponse=%s", string(allowedPretty))

	dataPretty, err := json.MarshalIndent(dataResp, "", "  ")
	if err != nil {
		logging.Fatalf("9efef5d2-f500-450f-929f-890f4d89f777", "failed to marshal data response: %v", err)
	}
	logging.Infof("70e0f6f2-72fd-42bf-9f0e-f9afca6ebc52", "DataSetResponse=%s", string(dataPretty))
}
