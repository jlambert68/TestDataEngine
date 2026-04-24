package webapi

import (
	"encoding/json"

	"TestDataEngine/internal/filters"
)

func BuildMetadataRequest(cfg DataSourceConfig) filters.FilterRequest {
	return filters.FilterRequest{
		SchemaVersion:  "1.0",
		RequestUUID:    "11111111-1111-4111-8111-111111111111",
		DataSourceUUID: cfg.DataSourceUUID,
		DataSourceName: cfg.DataSourceName,
		RequestFilter:  json.RawMessage(`{"field":"TestDataId","op":"exists","value":true}`),
	}
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
