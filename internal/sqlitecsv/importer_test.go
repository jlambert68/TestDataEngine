package sqlitecsv

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"regexp"
	"testing"
)

// TestImportRawCSV validates CSV-to-SQLite import using a temporary database.
func TestImportRawCSV(t *testing.T) {
	t.Parallel()

	tmp := t.TempDir()
	dbPath := filepath.Join(tmp, "data.sqlite")
	csvPath := filepath.Join(tmp, "raw.csv")

	if err := os.WriteFile(csvPath, []byte("\ufeffColA;ColB\n1;x\n2;y\n"), 0o644); err != nil {
		t.Fatalf("write csv: %v", err)
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()

	schema := `
create table main.data_items
(
    DataSourceUuid      TEXT not null,
    DataSourceName      TEXT not null,
    TestDataDomainUuid  TEXT not null,
    TestDataDomainName  TEXT not null,
    TestDataSourceTemplateName TEXT not null,
    DataUuid            TEXT not null,
    DataUpdateTimeStamp TEXT not null,
    JsonDataUuid        TEXT not null primary key,
    JsonData            TEXT not null,
    check (json_valid(JsonData))
);`
	if _, err := db.Exec(schema); err != nil {
		t.Fatalf("create schema: %v", err)
	}

	result, err := ImportRawCSV(context.Background(), ImportOptions{
		DBPath:         dbPath,
		CSVPath:        csvPath,
		DataSourceUUID: "110cc994-a913-4041-96fe-a96d7e0c97e8",
		DataSourceName: "SubCustody",
		TableName:      "main.data_items",
		Delimiter:      ';',
	})
	if err != nil {
		t.Fatalf("ImportRawCSV failed: %v", err)
	}
	if result.RowsInserted != 2 {
		t.Fatalf("expected 2 inserted rows, got %d", result.RowsInserted)
	}

	var count int
	if err := db.QueryRow(`select count(*) from main.data_items`).Scan(&count); err != nil {
		t.Fatalf("count rows: %v", err)
	}
	if count != 2 {
		t.Fatalf("expected 2 rows in db, got %d", count)
	}

	var dsUUID, dsName, domainUUID, domainName, templateName, valueA, valueB string
	query := `
select
  DataSourceUuid,
  DataSourceName,
  TestDataDomainUuid,
  TestDataDomainName,
  TestDataSourceTemplateName,
  json_extract(JsonData, '$.ColA'),
  json_extract(JsonData, '$.ColB')
from main.data_items
order by json_extract(JsonData, '$.ColA')
limit 1`
	if err := db.QueryRow(query).Scan(&dsUUID, &dsName, &domainUUID, &domainName, &templateName, &valueA, &valueB); err != nil {
		t.Fatalf("query row payload: %v", err)
	}
	if dsUUID != "110cc994-a913-4041-96fe-a96d7e0c97e8" || dsName != "SubCustody" {
		t.Fatalf("unexpected datasource metadata: %q %q", dsUUID, dsName)
	}
	if domainUUID != "7edf2269-a8d3-472c-aed6-8cdcc4a8b6ae" || domainName != "Sub Custody" || templateName != "SubCustodyMain" {
		t.Fatalf("unexpected domain metadata: %q %q %q", domainUUID, domainName, templateName)
	}
	if valueA != "1" || valueB != "x" {
		t.Fatalf("unexpected json payload values: ColA=%q ColB=%q", valueA, valueB)
	}
}

// TestImportRawCSVValidationErrors verifies input validation and missing-file behavior.
func TestImportRawCSVValidationErrors(t *testing.T) {
	t.Parallel()

	_, err := ImportRawCSV(context.Background(), ImportOptions{})
	if err == nil {
		t.Fatal("expected validation error")
	}

	_, err = ImportRawCSV(context.Background(), ImportOptions{
		DBPath:         "x.sqlite",
		CSVPath:        "x.csv",
		DataSourceUUID: "u",
		DataSourceName: "n",
	})
	if err == nil {
		t.Fatal("expected validation error for missing test data metadata before file open")
	}
	if err.Error() != "test data domain uuid is required" {
		t.Fatalf("unexpected validation error: %v", err)
	}

	_, err = ImportRawCSV(context.Background(), ImportOptions{
		DBPath:                     "x.sqlite",
		CSVPath:                    "x.csv",
		DataSourceUUID:             "u",
		DataSourceName:             "n",
		TestDataDomainUUID:         "d",
		TestDataDomainName:         "domain",
		TestDataSourceTemplateName: "template",
	})
	if err == nil {
		t.Fatal("expected open csv error for non-existing file")
	}
}

// TestInternalHelpers covers deterministic helper behavior used by the importer.
func TestInternalHelpers(t *testing.T) {
	t.Parallel()

	headers := normalizeHeaders([]string{"\ufeff A ", " B "})
	if len(headers) != 2 || headers[0] != "A" || headers[1] != "B" {
		t.Fatalf("unexpected headers: %#v", headers)
	}

	row := normalizeRecord([]string{"x"}, 3)
	if len(row) != 3 || row[0] != "x" || row[1] != "" || row[2] != "" {
		t.Fatalf("unexpected normalized row: %#v", row)
	}

	payload := buildPayload([]string{"a", "b"}, []string{" 1 ", " x "})
	if payload["a"] != "1" || payload["b"] != "x" {
		t.Fatalf("unexpected payload: %#v", payload)
	}

	id, err := newUUID()
	if err != nil {
		t.Fatalf("newUUID unexpected error: %v", err)
	}
	re := regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)
	if !re.MatchString(id) {
		t.Fatalf("unexpected uuid format: %q", id)
	}
}
