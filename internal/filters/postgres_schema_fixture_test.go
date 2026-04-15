package filters

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPostgresSchemaFixtureContainsRequiredColumns(t *testing.T) {
	t.Parallel()

	path := filepath.Join("..", "..", "testdata", "PostgresDB", "schema.sql")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read postgres schema fixture: %v", err)
	}

	text := string(raw)
	required := []string{
		"create table public.data_items",
		"DataSourceUuid",
		"DataSourceName",
		"TestDataDomainUuid",
		"TestDataDomainName",
		"TestDataSourceTemplateName",
		"DataUuid",
		"DataUpdateTimeStamp",
		"JsonDataUuid",
		"JsonData",
		"create table public.testdataset_response_schemas",
		"TestDataSourceName",
		"TestDataSourceUuid",
		"JsonSchemaName",
		"JsonSchema",
		"UpdatedDateTime",
	}
	for _, needle := range required {
		if !strings.Contains(text, needle) {
			t.Fatalf("postgres schema fixture missing %q", needle)
		}
	}
}
