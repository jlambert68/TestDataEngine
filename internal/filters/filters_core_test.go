package filters

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

func TestToSetAndOperatorSorting(t *testing.T) {
	ops := toSet("eq", "neq", "eq", "contains")
	logUnitCall(t, "11111111-1111-4111-8111-111111111111", "toSet", []string{"eq", "neq", "eq", "contains"}, "unique operator set", ops)
	if len(ops) != 3 {
		t.Fatalf("expected 3 unique operators, got %d", len(ops))
	}

	sorted := sortedOperators(ops)
	wantSorted := []string{"eq", "neq", "contains"}
	logUnitCall(t, "11111111-1111-4111-8111-111111111111", "sortedOperators", ops, wantSorted, sorted)
	if !reflect.DeepEqual(sorted, wantSorted) {
		t.Fatalf("unexpected sorted operators: got %v want %v", sorted, wantSorted)
	}

	fields := []AllowedFieldResult{
		{FieldName: "z"},
		{FieldName: "a"},
		{FieldName: "m"},
	}
	sortAllowedFields(fields)
	if got := []string{fields[0].FieldName, fields[1].FieldName, fields[2].FieldName}; !reflect.DeepEqual(got, []string{"a", "m", "z"}) {
		t.Fatalf("unexpected sorted field names: %v", got)
	}
	logUnitCall(t, "11111111-1111-4111-8111-111111111111", "sortAllowedFields", []string{"z", "a", "m"}, []string{"a", "m", "z"}, []string{fields[0].FieldName, fields[1].FieldName, fields[2].FieldName})
}

func TestValidateRequestAndCompileRequest(t *testing.T) {
	req := FilterRequest{
		SchemaVersion:  "1.0",
		RequestUUID:    "110cc994-a913-4041-96fe-a96d7e0c97e8",
		DataSourceUUID: "220cc994-a913-4041-96fe-a96d7e0c97e8",
		DataSourceName: "Orders",
		RequestFilter:  json.RawMessage(`{"field":"status","op":"eq","value":"active"}`),
	}

	if err := ValidateRequest(req); err != nil {
		t.Fatalf("ValidateRequest unexpected error: %v", err)
	}
	logUnitCall(t, "11111111-1111-4111-8111-111111111111", "ValidateRequest", req, "nil error", nil)

	compiled, err := CompileRequest(req)
	logUnitCall(t, "11111111-1111-4111-8111-111111111111", "CompileRequest", req, `WhereSQL=("status" = ?), Args=["active"]`, compiled)
	if err != nil {
		t.Fatalf("CompileRequest unexpected error: %v", err)
	}
	if compiled.WhereSQL != `("status" = ?)` {
		t.Fatalf("unexpected where sql: %s", compiled.WhereSQL)
	}
	if len(compiled.Args) != 1 || compiled.Args[0] != "active" {
		t.Fatalf("unexpected args: %#v", compiled.Args)
	}
}

func TestValidateRequestErrors(t *testing.T) {
	base := FilterRequest{
		SchemaVersion:  "1.0",
		RequestUUID:    "110cc994-a913-4041-96fe-a96d7e0c97e8",
		DataSourceUUID: "220cc994-a913-4041-96fe-a96d7e0c97e8",
		DataSourceName: "Orders",
		RequestFilter:  json.RawMessage(`{"field":"status","op":"eq","value":"active"}`),
	}

	cases := []struct {
		name    string
		mutate  func(*FilterRequest)
		wantErr string
	}{
		{
			name: "bad schema version",
			mutate: func(r *FilterRequest) {
				r.SchemaVersion = "2.0"
			},
			wantErr: "unsupported SchemaVersion",
		},
		{
			name: "bad request uuid",
			mutate: func(r *FilterRequest) {
				r.RequestUUID = "bad"
			},
			wantErr: "invalid RequestUuid",
		},
		{
			name: "bad datasource uuid",
			mutate: func(r *FilterRequest) {
				r.DataSourceUUID = "bad"
			},
			wantErr: "invalid DataSourceUuid",
		},
		{
			name: "missing datasource name",
			mutate: func(r *FilterRequest) {
				r.DataSourceName = ""
			},
			wantErr: "DataSourceName is required",
		},
		{
			name: "missing filter",
			mutate: func(r *FilterRequest) {
				r.RequestFilter = nil
			},
			wantErr: "RequestFilter is required",
		},
		{
			name: "unknown datasource",
			mutate: func(r *FilterRequest) {
				r.DataSourceName = "Unknown"
			},
			wantErr: `unknown DataSourceName: "Unknown"`,
		},
		{
			name: "uuid mismatch",
			mutate: func(r *FilterRequest) {
				r.DataSourceUUID = "11111111-1111-1111-1111-111111111111"
			},
			wantErr: "DataSourceName/DataSourceUuid mismatch",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := base
			tc.mutate(&req)
			err := ValidateRequest(req)
			logUnitCall(t, "11111111-1111-4111-8111-111111111111", "ValidateRequest", req, tc.wantErr, err)
			if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("expected error containing %q, got %v", tc.wantErr, err)
			}
		})
	}
}

func TestGetAllowedFieldsResponse(t *testing.T) {
	resp, err := GetAllowedFieldsResponse(
		"1.0",
		"110cc994-a913-4041-96fe-a96d7e0c97e8",
		"220cc994-a913-4041-96fe-a96d7e0c97e8",
		"Orders",
	)
	logUnitCall(t, "11111111-1111-4111-8111-111111111111", "GetAllowedFieldsResponse", map[string]any{
		"schemaVersion":  "1.0",
		"requestUUID":    "110cc994-a913-4041-96fe-a96d7e0c97e8",
		"dataSourceUUID": "220cc994-a913-4041-96fe-a96d7e0c97e8",
		"dataSourceName": "Orders",
	}, "non-empty AllowedFields", resp)
	if err != nil {
		t.Fatalf("GetAllowedFieldsResponse unexpected error: %v", err)
	}
	if len(resp.AllowedFields) == 0 {
		t.Fatal("expected allowed fields")
	}
	if resp.AllowedFields[0].FieldName != "amount" {
		t.Fatalf("expected sorted allowed fields, got first %q", resp.AllowedFields[0].FieldName)
	}

	_, err = GetAllowedFieldsResponse("2.0", "110cc994-a913-4041-96fe-a96d7e0c97e8", "220cc994-a913-4041-96fe-a96d7e0c97e8", "Orders")
	logUnitCall(t, "11111111-1111-4111-8111-111111111111", "GetAllowedFieldsResponse", "schemaVersion=2.0", "error", err)
	if err == nil {
		t.Fatal("expected error for invalid schema version")
	}
}

func TestCompileExpressionLogicalAndNot(t *testing.T) {
	fields := map[string]FieldDefinition{
		"status": {
			FieldType:          "string",
			SupportedOperators: toSet("eq"),
		},
		"priority": {
			FieldType:          "boolean",
			SupportedOperators: toSet("eq"),
		},
	}

	raw := json.RawMessage(`{
		"and": [
			{"field":"status","op":"eq","value":"active"},
			{"not":{"field":"priority","op":"eq","value":true}}
		]
	}`)
	sql, args, err := compileExpression(raw, fields)
	logUnitCall(t, "11111111-1111-4111-8111-111111111111", "compileExpression", string(raw), "non-empty SQL and 2 args", map[string]any{"sql": sql, "args": args, "err": err})
	if err != nil {
		t.Fatalf("compileExpression unexpected error: %v", err)
	}
	if sql == "" || len(args) != 2 {
		t.Fatalf("unexpected compile result sql=%q args=%v", sql, args)
	}

	_, _, err = compileLogical("AND", nil, fields)
	logUnitCall(t, "11111111-1111-4111-8111-111111111111", "compileLogical", map[string]any{"op": "AND", "parts": nil}, "error", err)
	if err == nil {
		t.Fatal("expected error for empty logical expression")
	}
}

func TestCompileComparisonOperators(t *testing.T) {
	fields := map[string]FieldDefinition{
		"s": {FieldType: "string", SupportedOperators: toSet("eq", "neq", "contains", "startsWith", "endsWith", "in", "nin", "exists", "isNull")},
		"n": {FieldType: "number", SupportedOperators: toSet("eq", "neq", "gt", "gte", "lt", "lte", "in", "nin", "exists", "isNull")},
		"b": {FieldType: "boolean", SupportedOperators: toSet("eq", "neq", "exists", "isNull")},
	}

	cases := []struct {
		name string
		cmp  Comparison
	}{
		{name: "eq", cmp: Comparison{Field: "s", Op: "eq", Value: "x"}},
		{name: "neq", cmp: Comparison{Field: "s", Op: "neq", Value: "x"}},
		{name: "gt", cmp: Comparison{Field: "n", Op: "gt", Value: 1.0}},
		{name: "gte", cmp: Comparison{Field: "n", Op: "gte", Value: 1.0}},
		{name: "lt", cmp: Comparison{Field: "n", Op: "lt", Value: 1.0}},
		{name: "lte", cmp: Comparison{Field: "n", Op: "lte", Value: 1.0}},
		{name: "contains", cmp: Comparison{Field: "s", Op: "contains", Value: "x"}},
		{name: "startsWith", cmp: Comparison{Field: "s", Op: "startsWith", Value: "x"}},
		{name: "endsWith", cmp: Comparison{Field: "s", Op: "endsWith", Value: "x"}},
		{name: "exists", cmp: Comparison{Field: "b", Op: "exists", Value: true}},
		{name: "isNull", cmp: Comparison{Field: "b", Op: "isNull", Value: false}},
		{name: "in", cmp: Comparison{Field: "s", Op: "in", Value: []interface{}{"a", "b"}}},
		{name: "nin", cmp: Comparison{Field: "n", Op: "nin", Value: []interface{}{1.0, 2.0}}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			sql, args, err := compileComparison(tc.cmp, fields)
			logUnitCall(t, "11111111-1111-4111-8111-111111111111", "compileComparison", tc.cmp, "sql and args", map[string]any{"sql": sql, "args": args, "err": err})
			if err != nil {
				t.Fatalf("compileComparison unexpected error: %v", err)
			}
			if sql == "" {
				t.Fatal("expected sql")
			}
			if tc.cmp.Op == "exists" || tc.cmp.Op == "isNull" {
				return
			}
			if len(args) == 0 {
				t.Fatalf("expected args for op %s", tc.cmp.Op)
			}
		})
	}

	_, _, err := compileComparison(Comparison{Field: "bad-field", Op: "eq", Value: "x"}, fields)
	logUnitCall(t, "11111111-1111-4111-8111-111111111111", "compileComparison", Comparison{Field: "bad-field", Op: "eq", Value: "x"}, "unsafe field error", err)
	if err == nil || !strings.Contains(err.Error(), "unsafe field name") {
		t.Fatalf("expected unsafe field name error, got %v", err)
	}
}

func TestValidationAndHelperFunctions(t *testing.T) {
	if err := validateScalarValue("x", "string"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	logUnitCall(t, "11111111-1111-4111-8111-111111111111", "validateScalarValue", map[string]any{"v": "x", "fieldType": "string"}, "nil error", nil)
	if err := validateScalarValue(1.2, "number"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	logUnitCall(t, "11111111-1111-4111-8111-111111111111", "validateScalarValue", map[string]any{"v": 1.2, "fieldType": "number"}, "nil error", nil)
	if err := validateScalarValue(1.0, "integer"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	logUnitCall(t, "11111111-1111-4111-8111-111111111111", "validateScalarValue", map[string]any{"v": 1.0, "fieldType": "integer"}, "nil error", nil)
	if err := validateScalarValue(true, "boolean"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	logUnitCall(t, "11111111-1111-4111-8111-111111111111", "validateScalarValue", map[string]any{"v": true, "fieldType": "boolean"}, "nil error", nil)
	if err := validateScalarValue("x", "unknown"); err == nil {
		t.Fatal("expected unsupported type error")
	}
	logUnitCall(t, "11111111-1111-4111-8111-111111111111", "validateScalarValue", map[string]any{"v": "x", "fieldType": "unknown"}, "error", "error")

	if err := validateComparableValue(1.0, "number"); err != nil {
		t.Fatalf("unexpected compare validation error: %v", err)
	}
	logUnitCall(t, "11111111-1111-4111-8111-111111111111", "validateComparableValue", map[string]any{"v": 1.0, "fieldType": "number"}, "nil error", nil)
	if err := validateComparableValue("x", "string"); err == nil {
		t.Fatal("expected non-comparable type error")
	}
	logUnitCall(t, "11111111-1111-4111-8111-111111111111", "validateComparableValue", map[string]any{"v": "x", "fieldType": "string"}, "error", "error")

	arr, err := normalizeArray([]interface{}{1.0, 2.0})
	logUnitCall(t, "11111111-1111-4111-8111-111111111111", "normalizeArray", []interface{}{1.0, 2.0}, "len=2", arr)
	if err != nil || len(arr) != 2 {
		t.Fatalf("unexpected normalizeArray result: %v, %v", arr, err)
	}
	if _, err := normalizeArray("bad"); err == nil {
		t.Fatal("expected normalizeArray error")
	}
	logUnitCall(t, "11111111-1111-4111-8111-111111111111", "normalizeArray", "bad", "error", "error")

	if !isSafeIdentifier("field_1") || isSafeIdentifier("field-1") {
		t.Fatal("isSafeIdentifier returned unexpected result")
	}
	logUnitCall(t, "11111111-1111-4111-8111-111111111111", "isSafeIdentifier", []string{"field_1", "field-1"}, []bool{true, false}, []bool{isSafeIdentifier("field_1"), isSafeIdentifier("field-1")})
	if quoteIdentifier("abc") != `"abc"` {
		t.Fatalf("unexpected quoted identifier: %s", quoteIdentifier("abc"))
	}
	logUnitCall(t, "11111111-1111-4111-8111-111111111111", "quoteIdentifier", "abc", `"abc"`, quoteIdentifier("abc"))
	if !isUUID("110cc994-a913-4041-96fe-a96d7e0c97e8") || isUUID("not-uuid") {
		t.Fatal("isUUID returned unexpected result")
	}
	logUnitCall(t, "11111111-1111-4111-8111-111111111111", "isUUID", []string{"110cc994-a913-4041-96fe-a96d7e0c97e8", "not-uuid"}, []bool{true, false}, []bool{isUUID("110cc994-a913-4041-96fe-a96d7e0c97e8"), isUUID("not-uuid")})
	if !isJSONNumber(1.0) || isJSONNumber(1) {
		t.Fatal("isJSONNumber returned unexpected result")
	}
	logUnitCall(t, "11111111-1111-4111-8111-111111111111", "isJSONNumber", []interface{}{1.0, 1}, []bool{true, false}, []bool{isJSONNumber(1.0), isJSONNumber(1)})
	if !isJSONInteger(2.0) || isJSONInteger(2.5) || isJSONInteger("2") {
		t.Fatal("isJSONInteger returned unexpected result")
	}
	logUnitCall(t, "11111111-1111-4111-8111-111111111111", "isJSONInteger", []interface{}{2.0, 2.5, "2"}, []bool{true, false, false}, []bool{isJSONInteger(2.0), isJSONInteger(2.5), isJSONInteger("2")})
}
