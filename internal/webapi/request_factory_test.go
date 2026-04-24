package webapi

import (
	"encoding/json"
	"testing"

	"TestDataEngine/internal/filters"
)

func TestBuildMetadataRequestUsesSchemaDerivedField(t *testing.T) {
	t.Parallel()

	cfg := DataSourceConfig{
		DataSourceName: "SubCustody",
		DataSourceUUID: "110cc994-a913-4041-96fe-a96d7e0c97e8",
	}

	req, err := BuildMetadataRequest(cfg)
	if err != nil {
		t.Fatalf("BuildMetadataRequest unexpected error: %v", err)
	}

	catalog, err := filters.LoadSchemaFieldCatalog(specificDatasourceResponseSchemaPath)
	if err != nil {
		t.Fatalf("LoadSchemaFieldCatalog unexpected error: %v", err)
	}
	if len(catalog.Order) == 0 {
		t.Fatal("expected schema field order")
	}

	var probe struct {
		Field string `json:"field"`
		Op    string `json:"op"`
		Value bool   `json:"value"`
	}
	if err := json.Unmarshal(req.RequestFilter, &probe); err != nil {
		t.Fatalf("unmarshal RequestFilter: %v", err)
	}

	if probe.Field != catalog.Order[0] {
		t.Fatalf("expected probe field %q from schema, got %q", catalog.Order[0], probe.Field)
	}
	if probe.Op != "exists" || !probe.Value {
		t.Fatalf("unexpected probe filter: %#v", probe)
	}
}
