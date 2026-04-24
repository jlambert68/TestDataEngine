package filters

import (
	"path/filepath"
	"testing"
)

func TestLoadSchemaFieldCatalog(t *testing.T) {
	t.Parallel()

	catalog, err := LoadSchemaFieldCatalog(specificDatasourceSchemaPath)
	if err != nil {
		t.Fatalf("LoadSchemaFieldCatalog unexpected error: %v", err)
	}
	if catalog.SchemaName != filepath.Base(specificDatasourceSchemaPath) {
		t.Fatalf("unexpected schema name: %q", catalog.SchemaName)
	}
	if len(catalog.Fields) == 0 {
		t.Fatal("expected schema catalog fields")
	}

	accountCurrency, ok := catalog.Fields["AccountCurrency"]
	if !ok {
		t.Fatal("expected AccountCurrency field in schema catalog")
	}
	if accountCurrency.FieldType != "string" {
		t.Fatalf("unexpected AccountCurrency type: %q", accountCurrency.FieldType)
	}
	if accountCurrency.Nullable {
		t.Fatal("expected AccountCurrency to be non-nullable")
	}
}

func TestParseSchemaFieldCatalogNullableFields(t *testing.T) {
	t.Parallel()

	catalog, err := ParseSchemaFieldCatalog([]byte(`{
	  "$defs": {
	    "TestDataSetItem": {
	      "type": "object",
	      "required": ["A"],
	      "properties": {
	        "A": {"type": "string"},
	        "B": {"oneOf": [{"type":"null"}, {"type":"integer"}]}
	      }
	    }
	  }
	}`), "inline.json")
	if err != nil {
		t.Fatalf("ParseSchemaFieldCatalog unexpected error: %v", err)
	}

	if got := catalog.Order[0]; got != "A" {
		t.Fatalf("expected required field A to be first, got %q", got)
	}

	fieldB := catalog.Fields["B"]
	if fieldB.FieldType != "integer" {
		t.Fatalf("unexpected B type: %q", fieldB.FieldType)
	}
	if !fieldB.Nullable {
		t.Fatal("expected B to be nullable")
	}
}

func TestBuildFieldDefinitionsFromCatalog(t *testing.T) {
	t.Parallel()

	catalog := SchemaFieldCatalog{
		Fields: map[string]SchemaField{
			"AccountCurrency": {
				CanonicalName:      "AccountCurrency",
				FieldType:          "string",
				Nullable:           false,
				SupportedOperators: toSet("eq", "contains"),
				Description:        "currency",
			},
		},
	}
	fields := BuildFieldDefinitionsFromCatalog(catalog)
	def, ok := fields["AccountCurrency"]
	if !ok {
		t.Fatal("expected built field definition")
	}
	if def.FieldType != "string" || def.Nullable {
		t.Fatalf("unexpected field definition: %#v", def)
	}
	if _, ok := def.SupportedOperators["contains"]; !ok {
		t.Fatalf("expected supported operator copy: %#v", def.SupportedOperators)
	}
}

func TestNormalizeRowToCanonical(t *testing.T) {
	t.Parallel()

	catalog, err := loadSpecificDatasourceSchemaCatalog()
	if err != nil {
		t.Fatalf("loadSpecificDatasourceSchemaCatalog unexpected error: %v", err)
	}

	row := map[string]interface{}{
		"ClientJuristictionCountryCode": "SE",
		"AccountCurrency":               "SEK",
	}
	got := NormalizeRowToCanonical(row, catalog)
	if _, ok := got["ClientJuristictionCountryCode"]; ok {
		t.Fatalf("expected raw alias to be removed: %#v", got)
	}
	if got["ClientJurisdictionCountryCode"] != "SE" {
		t.Fatalf("unexpected canonical alias value: %#v", got["ClientJurisdictionCountryCode"])
	}
	if got["AccountCurrency"] != "SEK" {
		t.Fatalf("unexpected untouched value: %#v", got["AccountCurrency"])
	}
}

func TestDescribeSQLiteDataSourceUsesSchemaFallback(t *testing.T) {
	t.Parallel()

	dbPath := filepath.Join(t.TempDir(), "source.sqlite")
	createSQLiteTestData(t, dbPath)

	req := FilterRequest{
		SchemaVersion:  "1.0",
		RequestUUID:    "6e6e17c4-6cc0-4ef0-a1cf-e96f0c5f8b8f",
		DataSourceUUID: "110cc994-a913-4041-96fe-a96d7e0c97e8",
		DataSourceName: "SubCustody",
	}

	resp, err := DescribeSQLiteDataSource(req, dbPath, "main.data_items")
	if err != nil {
		t.Fatalf("DescribeSQLiteDataSource unexpected error: %v", err)
	}
	if len(resp.AllowedFields) == 0 {
		t.Fatal("expected schema-driven allowed fields")
	}

	var found bool
	for _, field := range resp.AllowedFields {
		if field.FieldName == "ClientJurisdictionCountryCode" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected canonical schema field name in allowed fields: %#v", resp.AllowedFields)
	}
}
