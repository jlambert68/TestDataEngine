package filters

import "testing"

func TestQueryPostgresDataSourceErrors(t *testing.T) {
	t.Parallel()

	req := FilterRequest{
		SchemaVersion:  "1.0",
		RequestUUID:    "6e6e17c4-6cc0-4ef0-a1cf-e96f0c5f8b8f",
		DataSourceUUID: "110cc994-a913-4041-96fe-a96d7e0c97e8",
		DataSourceName: "SubCustody",
		RequestFilter:  []byte(`{"field":"AccountCurrency","op":"eq","value":"SEK"}`),
	}

	_, _, _, err := QueryPostgresDataSource(req, "", "public.data_items", postgresResponseSchemaTable, 1)
	if err == nil {
		t.Fatal("expected postgres dsn validation error")
	}

	_, _, _, err = QueryPostgresDataSource(req, "postgres://example", "public.data_items;drop table x", postgresResponseSchemaTable, 1)
	if err == nil {
		t.Fatal("expected unsafe postgres data table error")
	}

	_, _, _, err = QueryPostgresDataSource(req, "postgres://example", "public.data_items", "public.testdataset_response_schemas;drop table x", 1)
	if err == nil {
		t.Fatal("expected unsafe postgres schema metadata table error")
	}
}

func TestQuoteQualifiedIdentifier(t *testing.T) {
	t.Parallel()

	if got := quoteQualifiedIdentifier("public.data_items"); got != `"public"."data_items"` {
		t.Fatalf("unexpected quoted identifier: %q", got)
	}
	if got := quoteQualifiedIdentifier("FenixTestData.data_items"); got != `"FenixTestData"."data_items"` {
		t.Fatalf("unexpected mixed-case quoted identifier: %q", got)
	}
	if got := quoteQualifiedIdentifier("data_items"); got != `"data_items"` {
		t.Fatalf("unexpected single-part quoted identifier: %q", got)
	}
}
