package filters

import (
	"encoding/json"
	"path/filepath"
	"reflect"
	"testing"
)

// TestQueryCSVDataSourceMaxItemsBehavior checks maxItems behavior for bounded and unbounded queries.
func TestQueryCSVDataSourceMaxItemsBehavior(t *testing.T) {
	t.Parallel()

	req := FilterRequest{
		SchemaVersion:  "1.0",
		RequestUUID:    "6e6e17c4-6cc0-4ef0-a1cf-e96f0c5f8b8f",
		DataSourceUUID: "110cc994-a913-4041-96fe-a96d7e0c97e8",
		DataSourceName: "SubCustody",
		RequestFilter:  json.RawMessage(`{"field":"AccountEnvironment","op":"eq","value":"SysTest"}`),
	}
	csvPath := filepath.Join("..", "..", "P26_2", "FenixRawTestdata_646rows_211220_stripped.csv")

	_, _, allRowsResp, err := QueryCSVDataSource(req, csvPath, 0)
	logUnitCall(t, "11111111-1111-4111-8111-111111111111", "QueryCSVDataSource", map[string]any{"maxItems": 0}, "all matching rows", map[string]any{"rows": len(allRowsResp.Data), "err": err})
	if err != nil {
		t.Fatalf("QueryCSVDataSource(maxItems=0) unexpected error: %v", err)
	}
	if len(allRowsResp.Data) <= 1 {
		t.Fatalf("expected more than 1 matching row for broad filter, got %d", len(allRowsResp.Data))
	}

	_, _, negativeRowsResp, err := QueryCSVDataSource(req, csvPath, -1)
	logUnitCall(t, "11111111-1111-4111-8111-111111111111", "QueryCSVDataSource", map[string]any{"maxItems": -1}, "same behavior as maxItems=0", map[string]any{"rows": len(negativeRowsResp.Data), "err": err})
	if err != nil {
		t.Fatalf("QueryCSVDataSource(maxItems=-1) unexpected error: %v", err)
	}
	if len(negativeRowsResp.Data) != len(allRowsResp.Data) {
		t.Fatalf("expected maxItems=-1 to behave as unbounded; got %d vs %d", len(negativeRowsResp.Data), len(allRowsResp.Data))
	}

	_, _, oneRowResp, err := QueryCSVDataSource(req, csvPath, 1)
	logUnitCall(t, "11111111-1111-4111-8111-111111111111", "QueryCSVDataSource", map[string]any{"maxItems": 1}, "single row", map[string]any{"rows": len(oneRowResp.Data), "err": err})
	if err != nil {
		t.Fatalf("QueryCSVDataSource(maxItems=1) unexpected error: %v", err)
	}
	if len(oneRowResp.Data) != 1 {
		t.Fatalf("expected exactly 1 row when maxItems=1, got %d", len(oneRowResp.Data))
	}

	_, _, threeRowsResp, err := QueryCSVDataSource(req, csvPath, 3)
	logUnitCall(t, "11111111-1111-4111-8111-111111111111", "QueryCSVDataSource", map[string]any{"maxItems": 3}, "three rows", map[string]any{"rows": len(threeRowsResp.Data), "err": err})
	if err != nil {
		t.Fatalf("QueryCSVDataSource(maxItems=3) unexpected error: %v", err)
	}
	if len(threeRowsResp.Data) != 3 {
		t.Fatalf("expected exactly 3 rows when maxItems=3, got %d", len(threeRowsResp.Data))
	}
}

// TestShuffleRowsPreservesElements ensures shuffling changes order only, not content.
func TestShuffleRowsPreservesElements(t *testing.T) {
	rows := []map[string]interface{}{
		{"id": 1},
		{"id": 2},
		{"id": 3},
		{"id": 4},
	}
	original := make(map[int]struct{}, len(rows))
	for _, row := range rows {
		original[row["id"].(int)] = struct{}{}
	}

	shuffleRows(rows)
	logUnitCall(t, "11111111-1111-4111-8111-111111111111", "shuffleRows", "4 rows with ids 1..4", "same ids after shuffle", rows)

	if len(rows) != 4 {
		t.Fatalf("expected row count to stay 4, got %d", len(rows))
	}
	for _, row := range rows {
		id := row["id"].(int)
		if _, ok := original[id]; !ok {
			t.Fatalf("shuffleRows introduced unexpected id %d", id)
		}
	}

	single := []map[string]interface{}{{"id": 99}}
	shuffleRows(single)
	logUnitCall(t, "11111111-1111-4111-8111-111111111111", "shuffleRows", "single row", "unchanged single row", single)
	if single[0]["id"].(int) != 99 {
		t.Fatalf("expected single row to remain unchanged, got %#v", single[0])
	}
}

// TestShuffleRowsWithGUIDDeterministic verifies deterministic shuffle when seed GUID is fixed.
func TestShuffleRowsWithGUIDDeterministic(t *testing.T) {
	rowsA := []map[string]interface{}{
		{"id": 1},
		{"id": 2},
		{"id": 3},
		{"id": 4},
		{"id": 5},
	}
	rowsB := []map[string]interface{}{
		{"id": 1},
		{"id": 2},
		{"id": 3},
		{"id": 4},
		{"id": 5},
	}

	seedGUID := "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa"
	errA := shuffleRowsWithGUID(rowsA, seedGUID, 0)
	errB := shuffleRowsWithGUID(rowsB, seedGUID, 0)
	logUnitCall(t, "11111111-1111-4111-8111-111111111111", "shuffleRowsWithGUID", map[string]any{"seed": seedGUID, "run": "A"}, "nil error", errA)
	logUnitCall(t, "11111111-1111-4111-8111-111111111111", "shuffleRowsWithGUID", map[string]any{"seed": seedGUID, "run": "B"}, "nil error", errB)
	if errA != nil || errB != nil {
		t.Fatalf("unexpected shuffleRowsWithGUID errors: %v / %v", errA, errB)
	}
	if !reflect.DeepEqual(rowsA, rowsB) {
		t.Fatalf("expected deterministic shuffle for same seed; got A=%#v B=%#v", rowsA, rowsB)
	}
}

func TestSortRowsCanonicalBeforeSeededSelection(t *testing.T) {
	t.Parallel()

	ds := DataSourceDefinition{
		UUID: "110cc994-a913-4041-96fe-a96d7e0c97e8",
		Fields: map[string]FieldDefinition{
			"AccountCurrency":               {FieldType: "string", SupportedOperators: toSet("eq")},
			"AccountEnvironment":            {FieldType: "string", SupportedOperators: toSet("eq")},
			"ClientJuristictionCountryCode": {FieldType: "string", SupportedOperators: toSet("eq")},
			"TestDataId":                    {FieldType: "integer", SupportedOperators: toSet("eq")},
		},
	}
	req := FilterRequest{
		SchemaVersion:  "1.0",
		RequestUUID:    "6e6e17c4-6cc0-4ef0-a1cf-e96f0c5f8b8f",
		DataSourceUUID: "110cc994-a913-4041-96fe-a96d7e0c97e8",
		DataSourceName: "SubCustody",
		RequestFilter: json.RawMessage(`{
			"and": [
				{"field":"AccountCurrency","op":"eq","value":"SEK"},
				{"field":"AccountEnvironment","op":"eq","value":"SysTest"},
				{"field":"ClientJuristictionCountryCode","op":"eq","value":"SE"}
			]
		}`),
	}
	rowsA := []map[string]interface{}{
		{"TestDataId": int64(3), "AccountCurrency": "SEK", "AccountEnvironment": "SysTest", "ClientJuristictionCountryCode": "SE"},
		{"TestDataId": int64(1), "AccountCurrency": "SEK", "AccountEnvironment": "SysTest", "ClientJuristictionCountryCode": "SE"},
		{"TestDataId": int64(2), "AccountCurrency": "SEK", "AccountEnvironment": "SysTest", "ClientJuristictionCountryCode": "SE"},
	}
	rowsB := []map[string]interface{}{
		{"TestDataId": int64(1), "AccountCurrency": "SEK", "AccountEnvironment": "SysTest", "ClientJuristictionCountryCode": "SE"},
		{"TestDataId": int64(2), "AccountCurrency": "SEK", "AccountEnvironment": "SysTest", "ClientJuristictionCountryCode": "SE"},
		{"TestDataId": int64(3), "AccountCurrency": "SEK", "AccountEnvironment": "SysTest", "ClientJuristictionCountryCode": "SE"},
	}
	seedGUID := "cccccccc-cccc-4ccc-8ccc-cccccccccccc"

	_, _, respA, errA := queryDataRows(req, ds, rowsA, "testA", 1, seedGUID, 0, nil)
	_, _, respB, errB := queryDataRows(req, ds, rowsB, "testB", 1, seedGUID, 0, nil)
	if errA != nil || errB != nil {
		t.Fatalf("unexpected queryDataRows errors: %v / %v", errA, errB)
	}
	if !reflect.DeepEqual(respA.Data, respB.Data) {
		t.Fatalf("expected same seeded selection after canonical ordering; got A=%#v B=%#v", respA.Data, respB.Data)
	}
}

// TestQueryCSVDataSourceWithSeed verifies deterministic row selection for fixed seed GUID.
func TestQueryCSVDataSourceWithSeed(t *testing.T) {
	t.Parallel()

	req := FilterRequest{
		SchemaVersion:  "1.0",
		RequestUUID:    "6e6e17c4-6cc0-4ef0-a1cf-e96f0c5f8b8f",
		DataSourceUUID: "110cc994-a913-4041-96fe-a96d7e0c97e8",
		DataSourceName: "SubCustody",
		RequestFilter:  json.RawMessage(`{"field":"AccountEnvironment","op":"eq","value":"SysTest"}`),
	}
	csvPath := filepath.Join("..", "..", "P26_2", "FenixRawTestdata_646rows_211220_stripped.csv")
	seedGUID := "bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb"

	_, _, first, err1 := QueryCSVDataSourceWithSeed(req, csvPath, 5, seedGUID, 0)
	_, _, second, err2 := QueryCSVDataSourceWithSeed(req, csvPath, 5, seedGUID, 0)
	logUnitCall(t, "11111111-1111-4111-8111-111111111111", "QueryCSVDataSourceWithSeed", map[string]any{"seed": seedGUID, "run": "first"}, "nil error + deterministic rows", map[string]any{"err": err1, "rows": first.Data})
	logUnitCall(t, "11111111-1111-4111-8111-111111111111", "QueryCSVDataSourceWithSeed", map[string]any{"seed": seedGUID, "run": "second"}, "nil error + deterministic rows", map[string]any{"err": err2, "rows": second.Data})
	if err1 != nil || err2 != nil {
		t.Fatalf("unexpected QueryCSVDataSourceWithSeed errors: %v / %v", err1, err2)
	}
	if !reflect.DeepEqual(first.Data, second.Data) {
		t.Fatal("expected identical rows for repeated calls with same seed guid")
	}
}

// TestQueryCSVDataSourceWithSeedInvalidGUID verifies seed validation.
func TestQueryCSVDataSourceWithSeedInvalidGUID(t *testing.T) {
	t.Parallel()

	req := FilterRequest{
		SchemaVersion:  "1.0",
		RequestUUID:    "6e6e17c4-6cc0-4ef0-a1cf-e96f0c5f8b8f",
		DataSourceUUID: "110cc994-a913-4041-96fe-a96d7e0c97e8",
		DataSourceName: "SubCustody",
		RequestFilter:  json.RawMessage(`{"field":"AccountEnvironment","op":"eq","value":"SysTest"}`),
	}
	csvPath := filepath.Join("..", "..", "P26_2", "FenixRawTestdata_646rows_211220_stripped.csv")

	_, _, _, err := QueryCSVDataSourceWithSeed(req, csvPath, 2, "not-a-guid", 0)
	logUnitCall(t, "11111111-1111-4111-8111-111111111111", "QueryCSVDataSourceWithSeed", map[string]any{"seed": "not-a-guid"}, "invalid seed error", err)
	if err == nil {
		t.Fatal("expected invalid random seed guid error")
	}
}

func TestSeededRandOffsetBehavior(t *testing.T) {
	t.Parallel()

	seedGUID := "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa"

	rA, err := seededRand(seedGUID, 5)
	if err != nil {
		t.Fatalf("seededRand unexpected error: %v", err)
	}
	rB, err := seededRand(seedGUID, 5)
	if err != nil {
		t.Fatalf("seededRand unexpected error: %v", err)
	}
	if gotA, gotB := rA.Int63(), rB.Int63(); gotA != gotB {
		t.Fatalf("expected same deterministic stream for same guid+offset, got %d vs %d", gotA, gotB)
	}

	rOffset0, err := seededRand(seedGUID, 0)
	if err != nil {
		t.Fatalf("seededRand unexpected error: %v", err)
	}
	rOffset1, err := seededRand(seedGUID, 1)
	if err != nil {
		t.Fatalf("seededRand unexpected error: %v", err)
	}
	if got0, got1 := rOffset0.Int63(), rOffset1.Int63(); got0 == got1 {
		t.Fatalf("expected different deterministic streams for different offsets, got equal value %d", got0)
	}
}

func TestSeededRandOffsetValidation(t *testing.T) {
	t.Parallel()

	if _, err := seededRand("", 1); err == nil {
		t.Fatal("expected error when offset is provided without guid")
	}
	if _, err := seededRand("aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa", -1); err == nil {
		t.Fatal("expected error for negative offset")
	}
}
