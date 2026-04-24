package webapi

import (
	"encoding/json"
	"fmt"

	"TestDataEngine/internal/filters"
)

const specificDatasourceResponseSchemaPath = "internal/json/TestDataSet_Response_For_Specific_Datasource_From_TestDataEngine.json-schema.json"

func BuildMetadataRequest(cfg DataSourceConfig) (filters.FilterRequest, error) {
	schemaVersion, err := filters.RequestSchemaVersion()
	if err != nil {
		return filters.FilterRequest{}, fmt.Errorf("load request schema version: %w", err)
	}

	catalog, err := filters.LoadSchemaFieldCatalog(specificDatasourceResponseSchemaPath)
	if err != nil {
		return filters.FilterRequest{}, fmt.Errorf("load schema catalog: %w", err)
	}
	if len(catalog.Order) == 0 {
		return filters.FilterRequest{}, fmt.Errorf("schema catalog did not define any fields")
	}

	probeField := catalog.Order[0]
	requestFilter, err := json.Marshal(map[string]any{
		"field": probeField,
		"op":    "exists",
		"value": true,
	})
	if err != nil {
		return filters.FilterRequest{}, fmt.Errorf("marshal metadata filter: %w", err)
	}

	return filters.FilterRequest{
		SchemaVersion:  schemaVersion,
		RequestUUID:    "11111111-1111-4111-8111-111111111111",
		DataSourceUUID: cfg.DataSourceUUID,
		DataSourceName: cfg.DataSourceName,
		RequestFilter:  requestFilter,
	}, nil
}

func sourceTypeFromString(raw string) SourceType {
	switch SourceType(raw) {
	case SourceCSV, SourceSQLite, SourcePostgres:
		return SourceType(raw)
	default:
		return ""
	}
}

func sourceSupported(cfg DataSourceConfig, source SourceType) bool {
	for _, item := range cfg.SupportedSources {
		if item == source {
			return true
		}
	}
	return false
}
