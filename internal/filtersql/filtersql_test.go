package filtersql

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

type badExpression struct{}

func (badExpression) isExpression() {}

// TestOperatorValid verifies operator enum validation behavior.
func TestOperatorValid(t *testing.T) {
	validOps := []Operator{OpEq, OpNeq, OpGt, OpGte, OpLt, OpLte, OpIn, OpNin, OpContains, OpStartsWith, OpEndsWith, OpExists, OpIsNull}
	for _, op := range validOps {
		logUnitCall(t, "22222222-2222-4222-8222-222222222222", "Operator.Valid", op, true, op.Valid())
		if !op.Valid() {
			t.Fatalf("expected operator %q to be valid", op)
		}
	}
	logUnitCall(t, "22222222-2222-4222-8222-222222222222", "Operator.Valid", Operator("bad"), false, Operator("bad").Valid())
	if Operator("bad").Valid() {
		t.Fatal("expected unknown operator to be invalid")
	}
}

// TestMarkerMethods ensures expression marker methods satisfy the interface.
func TestMarkerMethods(t *testing.T) {
	Comparison{}.isExpression()
	AndExpression{}.isExpression()
	OrExpression{}.isExpression()
	NotExpression{}.isExpression()
	logUnitCall(t, "22222222-2222-4222-8222-222222222222", "isExpression markers", "Comparison/And/Or/Not", "no panic", "ok")
}

// TestParseExpressionAndHasOnlyKey covers JSON parsing into expression tree nodes.
func TestParseExpressionAndHasOnlyKey(t *testing.T) {
	logUnitCall(t, "22222222-2222-4222-8222-222222222222", "hasOnlyKey", map[string]json.RawMessage{"and": []byte(`[]`)}, true, hasOnlyKey(map[string]json.RawMessage{"and": []byte(`[]`)}, "and"))
	if !hasOnlyKey(map[string]json.RawMessage{"and": []byte(`[]`)}, "and") {
		t.Fatal("expected hasOnlyKey to be true")
	}
	logUnitCall(t, "22222222-2222-4222-8222-222222222222", "hasOnlyKey", map[string]json.RawMessage{"and": []byte(`[]`), "or": []byte(`[]`)}, false, hasOnlyKey(map[string]json.RawMessage{"and": []byte(`[]`), "or": []byte(`[]`)}, "and"))
	if hasOnlyKey(map[string]json.RawMessage{"and": []byte(`[]`), "or": []byte(`[]`)}, "and") {
		t.Fatal("expected hasOnlyKey to be false")
	}

	expr, err := ParseExpression([]byte(`{"and":[{"field":"age","op":"gte","value":18}]}`))
	logUnitCall(t, "22222222-2222-4222-8222-222222222222", "ParseExpression", `{"and":[...]}`, "AndExpression", map[string]any{"exprType": reflect.TypeOf(expr).String(), "err": err})
	if err != nil {
		t.Fatalf("ParseExpression(and) unexpected error: %v", err)
	}
	if _, ok := expr.(AndExpression); !ok {
		t.Fatalf("expected AndExpression, got %T", expr)
	}

	expr, err = ParseExpression([]byte(`{"or":[{"field":"city","op":"eq","value":"SE"}]}`))
	logUnitCall(t, "22222222-2222-4222-8222-222222222222", "ParseExpression", `{"or":[...]}`, "OrExpression", map[string]any{"exprType": reflect.TypeOf(expr).String(), "err": err})
	if err != nil {
		t.Fatalf("ParseExpression(or) unexpected error: %v", err)
	}
	if _, ok := expr.(OrExpression); !ok {
		t.Fatalf("expected OrExpression, got %T", expr)
	}

	expr, err = ParseExpression([]byte(`{"not":{"field":"x","op":"eq","value":"y"}}`))
	logUnitCall(t, "22222222-2222-4222-8222-222222222222", "ParseExpression", `{"not":...}`, "NotExpression", map[string]any{"exprType": reflect.TypeOf(expr).String(), "err": err})
	if err != nil {
		t.Fatalf("ParseExpression(not) unexpected error: %v", err)
	}
	if _, ok := expr.(NotExpression); !ok {
		t.Fatalf("expected NotExpression, got %T", expr)
	}

	expr, err = ParseExpression([]byte(`{"field":"x","op":"eq","value":"y"}`))
	logUnitCall(t, "22222222-2222-4222-8222-222222222222", "ParseExpression", `{"field":"x","op":"eq","value":"y"}`, "Comparison", map[string]any{"exprType": reflect.TypeOf(expr).String(), "err": err})
	if err != nil {
		t.Fatalf("ParseExpression(comparison) unexpected error: %v", err)
	}
	if _, ok := expr.(Comparison); !ok {
		t.Fatalf("expected Comparison, got %T", expr)
	}

	_, err = ParseExpression([]byte(`{}`))
	logUnitCall(t, "22222222-2222-4222-8222-222222222222", "ParseExpression", `{}`, "error", err)
	if err == nil {
		t.Fatal("expected parse error for empty object")
	}
	_, err = ParseExpression([]byte(`[]`))
	logUnitCall(t, "22222222-2222-4222-8222-222222222222", "ParseExpression", `[]`, "error", err)
	if err == nil {
		t.Fatal("expected parse error for non-object")
	}
}

// TestRequestUnmarshalJSON validates custom request unmarshalling with typed RequestFilter.
func TestRequestUnmarshalJSON(t *testing.T) {
	raw := []byte(`{
	  "SchemaVersion":"1.0",
	  "RequestUuid":"11111111-1111-4111-8111-111111111111",
	  "DataSourceUuid":"22222222-2222-4222-8222-222222222222",
	  "DataSourceName":"people",
	  "RequestFilter":{"field":"age","op":"gte","value":18}
	}`)

	var req Request
	if err := json.Unmarshal(raw, &req); err != nil {
		t.Fatalf("json.Unmarshal(Request) unexpected error: %v", err)
	}
	logUnitCall(t, "22222222-2222-4222-8222-222222222222", "json.Unmarshal(Request)", string(raw), "Request with Comparison filter", req)
	if req.SchemaVersion != "1.0" || req.DataSourceName != "people" {
		t.Fatalf("unexpected request values: %#v", req)
	}
	if _, ok := req.RequestFilter.(Comparison); !ok {
		t.Fatalf("expected comparison filter, got %T", req.RequestFilter)
	}

	var invalid Request
	err := json.Unmarshal([]byte(`{"RequestFilter":{}}`), &invalid)
	logUnitCall(t, "22222222-2222-4222-8222-222222222222", "json.Unmarshal(Request)", `{"RequestFilter":{}}`, "error", err)
	if err == nil || !strings.Contains(err.Error(), "invalid RequestFilter") {
		t.Fatalf("expected invalid RequestFilter error, got %v", err)
	}
}

// TestValidateRequestAndExpression validates request envelope and recursive expression checks.
func TestValidateRequestAndExpression(t *testing.T) {
	valid := Request{
		SchemaVersion:  SchemaVersion,
		RequestUUID:    "11111111-1111-4111-8111-111111111111",
		DataSourceUUID: "22222222-2222-4222-8222-222222222222",
		DataSourceName: "people",
		RequestFilter:  Comparison{Field: "age", Op: OpGte, Value: 18.0},
	}
	if err := ValidateRequest(valid); err != nil {
		t.Fatalf("ValidateRequest unexpected error: %v", err)
	}
	logUnitCall(t, "22222222-2222-4222-8222-222222222222", "ValidateRequest", valid, "nil error", nil)

	cases := []Request{
		{SchemaVersion: "2.0", RequestUUID: valid.RequestUUID, DataSourceUUID: valid.DataSourceUUID, DataSourceName: valid.DataSourceName, RequestFilter: valid.RequestFilter},
		{SchemaVersion: SchemaVersion, RequestUUID: "bad", DataSourceUUID: valid.DataSourceUUID, DataSourceName: valid.DataSourceName, RequestFilter: valid.RequestFilter},
		{SchemaVersion: SchemaVersion, RequestUUID: valid.RequestUUID, DataSourceUUID: "bad", DataSourceName: valid.DataSourceName, RequestFilter: valid.RequestFilter},
		{SchemaVersion: SchemaVersion, RequestUUID: valid.RequestUUID, DataSourceUUID: valid.DataSourceUUID, DataSourceName: "", RequestFilter: valid.RequestFilter},
		{SchemaVersion: SchemaVersion, RequestUUID: valid.RequestUUID, DataSourceUUID: valid.DataSourceUUID, DataSourceName: valid.DataSourceName, RequestFilter: nil},
	}
	for i, r := range cases {
		err := ValidateRequest(r)
		logUnitCall(t, "22222222-2222-4222-8222-222222222222", "ValidateRequest", r, "error", err)
		if err == nil {
			t.Fatalf("expected ValidateRequest error for case %d", i)
		}
	}

	err := ValidateExpression(AndExpression{And: []Expression{Comparison{Field: "x", Op: OpEq, Value: "y"}}})
	logUnitCall(t, "22222222-2222-4222-8222-222222222222", "ValidateExpression", "AndExpression", "nil error", err)
	if err != nil {
		t.Fatalf("ValidateExpression(and) unexpected error: %v", err)
	}
	err = ValidateExpression(OrExpression{Or: []Expression{Comparison{Field: "x", Op: OpEq, Value: "y"}}})
	logUnitCall(t, "22222222-2222-4222-8222-222222222222", "ValidateExpression", "OrExpression", "nil error", err)
	if err != nil {
		t.Fatalf("ValidateExpression(or) unexpected error: %v", err)
	}
	err = ValidateExpression(NotExpression{Not: Comparison{Field: "x", Op: OpEq, Value: "y"}})
	logUnitCall(t, "22222222-2222-4222-8222-222222222222", "ValidateExpression", "NotExpression", "nil error", err)
	if err != nil {
		t.Fatalf("ValidateExpression(not) unexpected error: %v", err)
	}
	err = ValidateExpression(AndExpression{})
	logUnitCall(t, "22222222-2222-4222-8222-222222222222", "ValidateExpression", "empty AndExpression", "error", err)
	if err == nil {
		t.Fatal("expected empty and error")
	}
	err = ValidateExpression(OrExpression{})
	logUnitCall(t, "22222222-2222-4222-8222-222222222222", "ValidateExpression", "empty OrExpression", "error", err)
	if err == nil {
		t.Fatal("expected empty or error")
	}
	err = ValidateExpression(NotExpression{})
	logUnitCall(t, "22222222-2222-4222-8222-222222222222", "ValidateExpression", "empty NotExpression", "error", err)
	if err == nil {
		t.Fatal("expected empty not error")
	}
	err = ValidateExpression(badExpression{})
	logUnitCall(t, "22222222-2222-4222-8222-222222222222", "ValidateExpression", "badExpression", "error", err)
	if err == nil {
		t.Fatal("expected unsupported expression type error")
	}
}

// TestValidateComparisonAndScalarHelpers checks comparison validation and scalar helper behavior.
func TestValidateComparisonAndScalarHelpers(t *testing.T) {
	comparisonCases := []Comparison{
		{Field: "x", Op: OpEq, Value: "v"},
		{Field: "x", Op: OpGt, Value: 1.0},
		{Field: "x", Op: OpContains, Value: "abc"},
		{Field: "x", Op: OpIn, Value: []any{"a", "b"}},
		{Field: "x", Op: OpExists, Value: true},
		{Field: "x", Op: OpIsNull, Value: false},
	}
	for _, cmp := range comparisonCases {
		err := validateComparison(cmp)
		logUnitCall(t, "22222222-2222-4222-8222-222222222222", "validateComparison", cmp, "nil error", err)
		if err != nil {
			t.Fatalf("validateComparison unexpected error for %q: %v", cmp.Op, err)
		}
	}

	errCases := []Comparison{
		{Field: "", Op: OpEq, Value: "x"},
		{Field: "x", Op: Operator("bad"), Value: "x"},
		{Field: "x", Op: OpExists, Value: "yes"},
		{Field: "x", Op: OpIsNull, Value: "yes"},
		{Field: "x", Op: OpIn, Value: []any{}},
		{Field: "x", Op: OpContains, Value: 1.0},
		{Field: "x", Op: OpEq, Value: nil},
		{Field: "x", Op: OpEq, Value: map[string]any{"x": "y"}},
	}
	for i, cmp := range errCases {
		err := validateComparison(cmp)
		logUnitCall(t, "22222222-2222-4222-8222-222222222222", "validateComparison", cmp, "error", err)
		if err == nil {
			t.Fatalf("expected validateComparison error for case %d", i)
		}
	}

	logUnitCall(t, "22222222-2222-4222-8222-222222222222", "isScalar", []any{nil, "x", 1.0, true}, []bool{true, true, true, true}, []bool{isScalar(nil), isScalar("x"), isScalar(1.0), isScalar(true)})
	if !isScalar(nil) || !isScalar("x") || !isScalar(1.0) || !isScalar(true) {
		t.Fatal("expected scalar values to pass")
	}
	logUnitCall(t, "22222222-2222-4222-8222-222222222222", "isScalar", []any{[]string{"x"}, map[string]any{"x": 1}}, []bool{false, false}, []bool{isScalar([]string{"x"}), isScalar(map[string]any{"x": 1})})
	if isScalar([]string{"x"}) || isScalar(map[string]any{"x": 1}) {
		t.Fatal("expected composite values to fail scalar check")
	}
}

// TestCompilerCompileAndInternals exercises compiler output and key internal helpers.
func TestCompilerCompileAndInternals(t *testing.T) {
	compiler := Compiler{Placeholder: Dollar, QuoteIdent: true}
	req := Request{
		SchemaVersion:  SchemaVersion,
		RequestUUID:    "11111111-1111-4111-8111-111111111111",
		DataSourceUUID: "22222222-2222-4222-8222-222222222222",
		DataSourceName: "people",
		RequestFilter: AndExpression{And: []Expression{
			Comparison{Field: "age", Op: OpGte, Value: 18.0},
			OrExpression{Or: []Expression{
				Comparison{Field: "city", Op: OpEq, Value: "Stockholm"},
				Comparison{Field: "city", Op: OpEq, Value: "Malmo"},
			}},
			NotExpression{Not: Comparison{Field: "name", Op: OpStartsWith, Value: "X"}},
		}},
	}

	where, args, err := compiler.Compile(req)
	logUnitCall(t, "22222222-2222-4222-8222-222222222222", "Compiler.Compile", req, "non-empty where and 4 args", map[string]any{"where": where, "args": args, "err": err})
	if err != nil {
		t.Fatalf("Compiler.Compile unexpected error: %v", err)
	}
	if where == "" || len(args) != 4 {
		t.Fatalf("unexpected compile output where=%q args=%v", where, args)
	}

	id, err := compiler.identifier("field_1")
	logUnitCall(t, "22222222-2222-4222-8222-222222222222", "Compiler.identifier", "field_1", `"field_1"`, map[string]any{"id": id, "err": err})
	if err != nil || id != `"field_1"` {
		t.Fatalf("unexpected identifier result: %q err=%v", id, err)
	}
	_, err = compiler.identifier("bad-field")
	logUnitCall(t, "22222222-2222-4222-8222-222222222222", "Compiler.identifier", "bad-field", "error", err)
	if err == nil {
		t.Fatal("expected invalid identifier error")
	}

	if got := compiler.ph(3); got != "$3" {
		t.Fatalf("unexpected dollar placeholder: %s", got)
	}
	logUnitCall(t, "22222222-2222-4222-8222-222222222222", "Compiler.ph", map[string]any{"placeholder": "Dollar", "n": 3}, "$3", compiler.ph(3))
	qCompiler := Compiler{Placeholder: Question}
	if got := qCompiler.ph(3); got != "?" {
		t.Fatalf("unexpected question placeholder: %s", got)
	}
	logUnitCall(t, "22222222-2222-4222-8222-222222222222", "Compiler.ph", map[string]any{"placeholder": "Question", "n": 3}, "?", qCompiler.ph(3))

	sql, args, err := compiler.compileComparison(Comparison{Field: "x", Op: OpEq, Value: nil}, 1)
	logUnitCall(t, "22222222-2222-4222-8222-222222222222", "Compiler.compileComparison", Comparison{Field: "x", Op: OpEq, Value: nil}, `"x" IS NULL`, map[string]any{"sql": sql, "args": args, "err": err})
	if err != nil || sql != `"x" IS NULL` || args != nil {
		t.Fatalf("unexpected eq nil compile result: %q %v %v", sql, args, err)
	}
	sql, args, err = compiler.compileComparison(Comparison{Field: "x", Op: OpNeq, Value: nil}, 1)
	logUnitCall(t, "22222222-2222-4222-8222-222222222222", "Compiler.compileComparison", Comparison{Field: "x", Op: OpNeq, Value: nil}, `"x" IS NOT NULL`, map[string]any{"sql": sql, "args": args, "err": err})
	if err != nil || sql != `"x" IS NOT NULL` || args != nil {
		t.Fatalf("unexpected neq nil compile result: %q %v %v", sql, args, err)
	}
	sql, args, err = compiler.compileComparison(Comparison{Field: "x", Op: OpIn, Value: []any{"a", "b"}}, 1)
	logUnitCall(t, "22222222-2222-4222-8222-222222222222", "Compiler.compileComparison", Comparison{Field: "x", Op: OpIn, Value: []any{"a", "b"}}, `"x" IN ($1, $2)`, map[string]any{"sql": sql, "args": args, "err": err})
	if err != nil {
		t.Fatalf("compileComparison(IN) unexpected error: %v", err)
	}
	if sql != `"x" IN ($1, $2)` || !reflect.DeepEqual(args, []any{"a", "b"}) {
		t.Fatalf("unexpected IN compile result: %q %v", sql, args)
	}
	sql, args, err = compiler.compileComparison(Comparison{Field: "x", Op: OpIsNull, Value: true}, 1)
	logUnitCall(t, "22222222-2222-4222-8222-222222222222", "Compiler.compileComparison", Comparison{Field: "x", Op: OpIsNull, Value: true}, `"x" IS NULL`, map[string]any{"sql": sql, "args": args, "err": err})
	if err != nil || sql != `"x" IS NULL` || args != nil {
		t.Fatalf("unexpected isNull compile result: %q %v %v", sql, args, err)
	}

	_, _, err = compiler.compileComparison(Comparison{Field: "x", Op: Operator("bad"), Value: "v"}, 1)
	logUnitCall(t, "22222222-2222-4222-8222-222222222222", "Compiler.compileComparison", Comparison{Field: "x", Op: Operator("bad"), Value: "v"}, "error", err)
	if err == nil {
		t.Fatal("expected unsupported operator error")
	}
	_, _, err = compiler.compileExpr(badExpression{}, 1)
	logUnitCall(t, "22222222-2222-4222-8222-222222222222", "Compiler.compileExpr", badExpression{}, "error", err)
	if err == nil {
		t.Fatal("expected unsupported expression error")
	}
}
