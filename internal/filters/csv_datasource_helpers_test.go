package filters

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestCompileAndAllowedFieldsForDataSource(t *testing.T) {
	ds := DataSourceDefinition{
		UUID: "110cc994-a913-4041-96fe-a96d7e0c97e8",
		Fields: map[string]FieldDefinition{
			"AccountCurrency": {FieldType: "string", SupportedOperators: toSet("eq"), Description: "currency"},
		},
	}
	req := FilterRequest{
		SchemaVersion:  "1.0",
		RequestUUID:    "6e6e17c4-6cc0-4ef0-a1cf-e96f0c5f8b8f",
		DataSourceUUID: ds.UUID,
		DataSourceName: "SubCustody",
		RequestFilter:  json.RawMessage(`{"field":"AccountCurrency","op":"eq","value":"SEK"}`),
	}

	compiled, err := compileRequestForDataSource(req, ds)
	logUnitCall(t, "11111111-1111-4111-8111-111111111111", "compileRequestForDataSource", map[string]any{"req": req, "ds": ds}, "compiled SQL", map[string]any{"compiled": compiled, "err": err})
	if err != nil {
		t.Fatalf("compileRequestForDataSource unexpected error: %v", err)
	}
	if compiled.WhereSQL == "" {
		t.Fatal("expected compiled sql")
	}

	allowed, err := allowedFieldsForDataSource(req, ds)
	logUnitCall(t, "11111111-1111-4111-8111-111111111111", "allowedFieldsForDataSource", map[string]any{"req": req, "ds": ds}, "1 allowed field", map[string]any{"allowed": allowed, "err": err})
	if err != nil {
		t.Fatalf("allowedFieldsForDataSource unexpected error: %v", err)
	}
	if len(allowed.AllowedFields) != 1 || allowed.AllowedFields[0].FieldName != "AccountCurrency" {
		t.Fatalf("unexpected allowed fields: %#v", allowed.AllowedFields)
	}

	badReq := req
	badReq.DataSourceUUID = "00000000-0000-0000-0000-000000000000"
	_, err = compileRequestForDataSource(badReq, ds)
	logUnitCall(t, "11111111-1111-4111-8111-111111111111", "compileRequestForDataSource", map[string]any{"req": badReq, "ds": ds}, "error", err)
	if err == nil {
		t.Fatal("expected datasource mismatch error")
	}
	_, err = allowedFieldsForDataSource(badReq, ds)
	logUnitCall(t, "11111111-1111-4111-8111-111111111111", "allowedFieldsForDataSource", map[string]any{"req": badReq, "ds": ds}, "error", err)
	if err == nil {
		t.Fatal("expected datasource mismatch error")
	}
}

func TestLoadCSVDataSource(t *testing.T) {
	t.Parallel()

	tmp := t.TempDir()
	csvPath := filepath.Join(tmp, "source.csv")
	content := "\ufeffA;B;C;D\n1;true;NULL;1,5\n2;false;X;2\n"
	if err := os.WriteFile(csvPath, []byte(content), 0o644); err != nil {
		t.Fatalf("write csv: %v", err)
	}

	ds, rows, err := loadCSVDataSource("110cc994-a913-4041-96fe-a96d7e0c97e8", csvPath)
	logUnitCall(t, "11111111-1111-4111-8111-111111111111", "loadCSVDataSource", map[string]any{"uuid": "110cc994-a913-4041-96fe-a96d7e0c97e8", "csvPath": csvPath}, "inferred fields + 2 rows", map[string]any{"fieldCount": len(ds.Fields), "rowCount": len(rows), "err": err})
	if err != nil {
		t.Fatalf("loadCSVDataSource unexpected error: %v", err)
	}
	if ds.Fields["A"].FieldType != "integer" {
		t.Fatalf("expected integer type for A, got %q", ds.Fields["A"].FieldType)
	}
	if ds.Fields["B"].FieldType != "boolean" {
		t.Fatalf("expected boolean type for B, got %q", ds.Fields["B"].FieldType)
	}
	if ds.Fields["C"].Nullable != true {
		t.Fatalf("expected C to be nullable")
	}
	if ds.Fields["D"].FieldType != "number" {
		t.Fatalf("expected number type for D, got %q", ds.Fields["D"].FieldType)
	}
	if len(rows) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(rows))
	}
	if rows[0]["C"] != nil {
		t.Fatalf("expected NULL to map to nil, got %#v", rows[0]["C"])
	}

	_, _, err = loadCSVDataSource("x", filepath.Join(tmp, "missing.csv"))
	logUnitCall(t, "11111111-1111-4111-8111-111111111111", "loadCSVDataSource", map[string]any{"uuid": "x", "csvPath": filepath.Join(tmp, "missing.csv")}, "open error", err)
	if err == nil {
		t.Fatal("expected open error for missing csv")
	}
}

func TestEvaluateExpressionAndComparison(t *testing.T) {
	fields := map[string]FieldDefinition{
		"s": {FieldType: "string", SupportedOperators: toSet("eq", "contains", "startsWith", "endsWith", "in", "nin", "neq")},
		"n": {FieldType: "number", SupportedOperators: toSet("eq", "gt", "gte", "lt", "lte", "in", "nin")},
		"b": {FieldType: "boolean", SupportedOperators: toSet("eq", "exists", "isNull", "neq")},
	}
	row := map[string]interface{}{
		"s": "hello",
		"n": 10.0,
		"b": true,
	}

	cases := []struct {
		name string
		raw  string
		want bool
	}{
		{name: "and", raw: `{"and":[{"field":"s","op":"eq","value":"hello"},{"field":"n","op":"gt","value":5}]}`, want: true},
		{name: "or", raw: `{"or":[{"field":"s","op":"eq","value":"x"},{"field":"n","op":"gt","value":5}]}`, want: true},
		{name: "not", raw: `{"not":{"field":"s","op":"eq","value":"x"}}`, want: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := evaluateExpression(json.RawMessage(tc.raw), row, fields)
			logUnitCall(t, "11111111-1111-4111-8111-111111111111", "evaluateExpression", map[string]any{"raw": tc.raw, "row": row}, tc.want, map[string]any{"got": got, "err": err})
			if err != nil {
				t.Fatalf("evaluateExpression unexpected error: %v", err)
			}
			if got != tc.want {
				t.Fatalf("unexpected evaluation result: got %v want %v", got, tc.want)
			}
		})
	}

	ops := []Comparison{
		{Field: "s", Op: "contains", Value: "ell"},
		{Field: "s", Op: "startsWith", Value: "he"},
		{Field: "s", Op: "endsWith", Value: "lo"},
		{Field: "b", Op: "exists", Value: true},
		{Field: "b", Op: "isNull", Value: false},
		{Field: "s", Op: "in", Value: []interface{}{"hello", "x"}},
		{Field: "n", Op: "nin", Value: []interface{}{1.0, 2.0}},
		{Field: "s", Op: "neq", Value: "x"},
		{Field: "n", Op: "lte", Value: 10.0},
		{Field: "n", Op: "gte", Value: 10.0},
		{Field: "n", Op: "lt", Value: 11.0},
	}
	for _, cmp := range ops {
		got, err := evaluateComparison(cmp, row, fields)
		logUnitCall(t, "11111111-1111-4111-8111-111111111111", "evaluateComparison", map[string]any{"comparison": cmp, "row": row}, true, map[string]any{"got": got, "err": err})
		if err != nil {
			t.Fatalf("evaluateComparison(%s) unexpected error: %v", cmp.Op, err)
		}
		if !got {
			t.Fatalf("evaluateComparison(%s) expected true", cmp.Op)
		}
	}

	_, err := evaluateExpression(json.RawMessage(`{}`), row, fields)
	logUnitCall(t, "11111111-1111-4111-8111-111111111111", "evaluateExpression", map[string]any{"raw": `{}`, "row": row}, "error", err)
	if err == nil {
		t.Fatal("expected invalid expression error")
	}
}

func TestCSVPrimitiveHelpers(t *testing.T) {
	err := validateRequestEnvelope(FilterRequest{})
	logUnitCall(t, "11111111-1111-4111-8111-111111111111", "validateRequestEnvelope", FilterRequest{}, "error", err)
	if err == nil {
		t.Fatal("expected validateRequestEnvelope error")
	}
	err = validateRequestEnvelope(FilterRequest{
		SchemaVersion:  "1.0",
		RequestUUID:    "6e6e17c4-6cc0-4ef0-a1cf-e96f0c5f8b8f",
		DataSourceUUID: "110cc994-a913-4041-96fe-a96d7e0c97e8",
		DataSourceName: "SubCustody",
		RequestFilter:  json.RawMessage(`{"field":"a","op":"eq","value":"x"}`),
	})
	logUnitCall(t, "11111111-1111-4111-8111-111111111111", "validateRequestEnvelope", "valid request envelope", "nil error", err)
	if err != nil {
		t.Fatalf("unexpected validateRequestEnvelope error: %v", err)
	}

	if got := normalizeHeaders([]string{"\ufeff A ", " B "}); !reflect.DeepEqual(got, []string{" A", "B"}) {
		t.Fatalf("unexpected normalizeHeaders output: %#v", got)
	}
	logUnitCall(t, "11111111-1111-4111-8111-111111111111", "normalizeHeaders", []string{"\\ufeff A ", " B "}, []string{" A", "B"}, normalizeHeaders([]string{"\ufeff A ", " B "}))
	if got := normalizeRecord([]string{"1"}, 3); len(got) != 3 {
		t.Fatalf("unexpected normalizeRecord length: %d", len(got))
	}
	logUnitCall(t, "11111111-1111-4111-8111-111111111111", "normalizeRecord", map[string]any{"record": []string{"1"}, "wantLen": 3}, "len=3", normalizeRecord([]string{"1"}, 3))

	if inferFieldType([]string{"true", "false", "NULL"}) != "boolean" {
		t.Fatal("expected boolean inference")
	}
	logUnitCall(t, "11111111-1111-4111-8111-111111111111", "inferFieldType", []string{"true", "false", "NULL"}, "boolean", inferFieldType([]string{"true", "false", "NULL"}))
	if inferFieldType([]string{"1", "2", ""}) != "integer" {
		t.Fatal("expected integer inference")
	}
	logUnitCall(t, "11111111-1111-4111-8111-111111111111", "inferFieldType", []string{"1", "2", ""}, "integer", inferFieldType([]string{"1", "2", ""}))
	if inferFieldType([]string{"1,5", "2", ""}) != "number" {
		t.Fatal("expected number inference")
	}
	logUnitCall(t, "11111111-1111-4111-8111-111111111111", "inferFieldType", []string{"1,5", "2", ""}, "number", inferFieldType([]string{"1,5", "2", ""}))
	if inferFieldType([]string{"x", "2"}) != "string" {
		t.Fatal("expected string inference")
	}
	logUnitCall(t, "11111111-1111-4111-8111-111111111111", "inferFieldType", []string{"x", "2"}, "string", inferFieldType([]string{"x", "2"}))

	if !hasNullValue([]string{"x", "NULL"}) || hasNullValue([]string{"x", "y"}) {
		t.Fatal("unexpected hasNullValue result")
	}
	logUnitCall(t, "11111111-1111-4111-8111-111111111111", "hasNullValue", []string{"x", "NULL"}, true, hasNullValue([]string{"x", "NULL"}))
	logUnitCall(t, "11111111-1111-4111-8111-111111111111", "hasNullValue", []string{"x", "y"}, false, hasNullValue([]string{"x", "y"}))
	if _, ok := supportedOperatorsForType("boolean")["exists"]; !ok {
		t.Fatal("expected exists operator for boolean")
	}
	logUnitCall(t, "11111111-1111-4111-8111-111111111111", "supportedOperatorsForType", "boolean", "contains exists", supportedOperatorsForType("boolean"))
	if _, ok := supportedOperatorsForType("string")["contains"]; !ok {
		t.Fatal("expected contains operator for string")
	}
	logUnitCall(t, "11111111-1111-4111-8111-111111111111", "supportedOperatorsForType", "string", "contains contains", supportedOperatorsForType("string"))

	val, err := parseCSVValue("1,5", "number")
	logUnitCall(t, "11111111-1111-4111-8111-111111111111", "parseCSVValue", map[string]any{"raw": "1,5", "fieldType": "number"}, 1.5, map[string]any{"val": val, "err": err})
	if err != nil || val.(float64) != 1.5 {
		t.Fatalf("unexpected parseCSVValue number: %#v err=%v", val, err)
	}
	val, err = parseCSVValue("12", "integer")
	logUnitCall(t, "11111111-1111-4111-8111-111111111111", "parseCSVValue", map[string]any{"raw": "12", "fieldType": "integer"}, int64(12), map[string]any{"val": val, "err": err})
	if err != nil || val.(int64) != 12 {
		t.Fatalf("unexpected parseCSVValue integer: %#v err=%v", val, err)
	}
	val, err = parseCSVValue("true", "boolean")
	logUnitCall(t, "11111111-1111-4111-8111-111111111111", "parseCSVValue", map[string]any{"raw": "true", "fieldType": "boolean"}, true, map[string]any{"val": val, "err": err})
	if err != nil || val.(bool) != true {
		t.Fatalf("unexpected parseCSVValue boolean: %#v err=%v", val, err)
	}
	val, err = parseCSVValue(" text ", "string")
	logUnitCall(t, "11111111-1111-4111-8111-111111111111", "parseCSVValue", map[string]any{"raw": " text ", "fieldType": "string"}, "text", map[string]any{"val": val, "err": err})
	if err != nil || val.(string) != "text" {
		t.Fatalf("unexpected parseCSVValue string: %#v err=%v", val, err)
	}
	val, err = parseCSVValue("NULL", "string")
	logUnitCall(t, "11111111-1111-4111-8111-111111111111", "parseCSVValue", map[string]any{"raw": "NULL", "fieldType": "string"}, nil, map[string]any{"val": val, "err": err})
	if err != nil || val != nil {
		t.Fatalf("expected NULL to parse as nil, got %#v err=%v", val, err)
	}

	if !isCSVNull("NULL") || !isCSVNull(" ") || isCSVNull("x") {
		t.Fatal("unexpected isCSVNull result")
	}
	logUnitCall(t, "11111111-1111-4111-8111-111111111111", "isCSVNull", []string{"NULL", " ", "x"}, []bool{true, true, false}, []bool{isCSVNull("NULL"), isCSVNull(" "), isCSVNull("x")})
	if b, err := parseBoolean("TRUE"); err != nil || !b {
		t.Fatalf("unexpected parseBoolean result: %v %v", b, err)
	}
	b, err := parseBoolean("TRUE")
	logUnitCall(t, "11111111-1111-4111-8111-111111111111", "parseBoolean", "TRUE", true, map[string]any{"value": b, "err": err})
	if _, err := parseBoolean("yes"); err == nil {
		t.Fatal("expected parseBoolean error")
	}
	_, err = parseBoolean("yes")
	logUnitCall(t, "11111111-1111-4111-8111-111111111111", "parseBoolean", "yes", "error", err)
	if n, err := parseInteger("1.0"); err != nil || n != 1 {
		t.Fatalf("unexpected parseInteger result: %v %v", n, err)
	}
	n, err := parseInteger("1.0")
	logUnitCall(t, "11111111-1111-4111-8111-111111111111", "parseInteger", "1.0", int64(1), map[string]any{"value": n, "err": err})
	if _, err := parseInteger("1.5"); err == nil {
		t.Fatal("expected parseInteger error")
	}
	_, err = parseInteger("1.5")
	logUnitCall(t, "11111111-1111-4111-8111-111111111111", "parseInteger", "1.5", "error", err)
	if f, err := parseNumber("1,5"); err != nil || f != 1.5 {
		t.Fatalf("unexpected parseNumber result: %v %v", f, err)
	}
	f, err := parseNumber("1,5")
	logUnitCall(t, "11111111-1111-4111-8111-111111111111", "parseNumber", "1,5", 1.5, map[string]any{"value": f, "err": err})

	for _, v := range []interface{}{float64(1), float32(1), int(1), int8(1), int16(1), int32(1), int64(1), uint(1), uint8(1), uint16(1), uint32(1), uint64(1)} {
		got, ok := toFloat(v)
		logUnitCall(t, "11111111-1111-4111-8111-111111111111", "toFloat", v, "ok=true", map[string]any{"value": got, "ok": ok})
		if !ok {
			t.Fatalf("expected toFloat success for type %T", v)
		}
	}
	gotFloat, ok := toFloat("x")
	logUnitCall(t, "11111111-1111-4111-8111-111111111111", "toFloat", "x", "ok=false", map[string]any{"value": gotFloat, "ok": ok})
	if ok {
		t.Fatal("expected toFloat false for string")
	}

	if eq, err := valuesEqual("number", 1.0, 1.0); err != nil || !eq {
		t.Fatalf("unexpected valuesEqual number: %v %v", eq, err)
	}
	eq, err := valuesEqual("number", 1.0, 1.0)
	logUnitCall(t, "11111111-1111-4111-8111-111111111111", "valuesEqual", map[string]any{"fieldType": "number", "rowValue": 1.0, "filterValue": 1.0}, true, map[string]any{"eq": eq, "err": err})
	if eq, err := valuesEqual("boolean", true, true); err != nil || !eq {
		t.Fatalf("unexpected valuesEqual boolean: %v %v", eq, err)
	}
	eq, err = valuesEqual("boolean", true, true)
	logUnitCall(t, "11111111-1111-4111-8111-111111111111", "valuesEqual", map[string]any{"fieldType": "boolean", "rowValue": true, "filterValue": true}, true, map[string]any{"eq": eq, "err": err})
	if eq, err := valuesEqual("string", "x", "x"); err != nil || !eq {
		t.Fatalf("unexpected valuesEqual string: %v %v", eq, err)
	}
	eq, err = valuesEqual("string", "x", "x")
	logUnitCall(t, "11111111-1111-4111-8111-111111111111", "valuesEqual", map[string]any{"fieldType": "string", "rowValue": "x", "filterValue": "x"}, true, map[string]any{"eq": eq, "err": err})
	_, err = valuesEqual("number", "x", 1.0)
	logUnitCall(t, "11111111-1111-4111-8111-111111111111", "valuesEqual", map[string]any{"fieldType": "number", "rowValue": "x", "filterValue": 1.0}, "error", err)
	if err == nil {
		t.Fatal("expected valuesEqual type error")
	}

	if ok, err := compareOrdered("number", 2.0, 1.0, func(left, right int) bool { return left > right }); err != nil || !ok {
		t.Fatalf("unexpected compareOrdered number: %v %v", ok, err)
	}
	okCmp, err := compareOrdered("number", 2.0, 1.0, func(left, right int) bool { return left > right })
	logUnitCall(t, "11111111-1111-4111-8111-111111111111", "compareOrdered", map[string]any{"fieldType": "number", "rowValue": 2.0, "filterValue": 1.0}, true, map[string]any{"ok": okCmp, "err": err})
	if ok, err := compareOrdered("string", "b", "a", func(left, right int) bool { return left > right }); err != nil || !ok {
		t.Fatalf("unexpected compareOrdered string: %v %v", ok, err)
	}
	okCmp, err = compareOrdered("string", "b", "a", func(left, right int) bool { return left > right })
	logUnitCall(t, "11111111-1111-4111-8111-111111111111", "compareOrdered", map[string]any{"fieldType": "string", "rowValue": "b", "filterValue": "a"}, true, map[string]any{"ok": okCmp, "err": err})
	_, err = compareOrdered("boolean", true, false, func(left, right int) bool { return left > right })
	logUnitCall(t, "11111111-1111-4111-8111-111111111111", "compareOrdered", map[string]any{"fieldType": "boolean", "rowValue": true, "filterValue": false}, "error", err)
	if err == nil {
		t.Fatal("expected compareOrdered unsupported type error")
	}
	if ok, err := compareOrdered("number", nil, 1.0, func(left, right int) bool { return left > right }); err != nil || ok {
		t.Fatalf("unexpected compareOrdered nil row value: %v %v", ok, err)
	}
	okCmp, err = compareOrdered("number", nil, 1.0, func(left, right int) bool { return left > right })
	logUnitCall(t, "11111111-1111-4111-8111-111111111111", "compareOrdered", map[string]any{"fieldType": "number", "rowValue": nil, "filterValue": 1.0}, false, map[string]any{"ok": okCmp, "err": err})

	if minInt(1, 2) != 1 || minInt(5, 2) != 2 {
		t.Fatal("unexpected minInt result")
	}
	logUnitCall(t, "11111111-1111-4111-8111-111111111111", "minInt", []int{1, 2}, 1, minInt(1, 2))
	logUnitCall(t, "11111111-1111-4111-8111-111111111111", "minInt", []int{5, 2}, 2, minInt(5, 2))

	_, _, _, err = QueryCSVDataSource(FilterRequest{}, "", 1)
	logUnitCall(t, "11111111-1111-4111-8111-111111111111", "QueryCSVDataSource", map[string]any{"request": FilterRequest{}, "csvPath": "", "maxItems": 1}, "error: unsupported SchemaVersion", err)
	if err == nil || !strings.Contains(err.Error(), "unsupported SchemaVersion") {
		t.Fatalf("expected request validation error, got %v", err)
	}
}
