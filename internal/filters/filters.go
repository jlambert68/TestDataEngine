package filters

import (
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
)

type FilterRequest struct {
	SchemaVersion  string          `json:"SchemaVersion"`
	RequestUUID    string          `json:"RequestUuid"`
	DataSourceUUID string          `json:"DataSourceUuid"`
	DataSourceName string          `json:"DataSourceName"`
	RequestFilter  json.RawMessage `json:"RequestFilter"`
}

// Comparison describes a single field/operator/value clause.
type Comparison struct {
	Field string      `json:"field"`
	Op    string      `json:"op"`
	Value interface{} `json:"value"`
}

// CompiledFilter is a SQL-like representation of RequestFilter.
type CompiledFilter struct {
	WhereSQL string
	Args     []interface{}
}

// FieldDefinition describes one datasource field and its filtering capabilities.
type FieldDefinition struct {
	FieldType          string
	Nullable           bool
	SupportedOperators map[string]struct{}
	Description        string
}

// AllowedFieldResponse is returned to clients that ask which fields/operators are valid.
type AllowedFieldResponse struct {
	SchemaVersion  string               `json:"SchemaVersion"`
	RequestUUID    string               `json:"RequestUuid"`
	DataSourceUUID string               `json:"DataSourceUuid"`
	DataSourceName string               `json:"DataSourceName"`
	AllowedFields  []AllowedFieldResult `json:"AllowedFields"`
}

type AllowedFieldResult struct {
	FieldName          string   `json:"FieldName"`
	FieldType          string   `json:"FieldType"`
	Nullable           bool     `json:"Nullable"`
	SupportedOperators []string `json:"SupportedOperators"`
	Description        string   `json:"Description,omitempty"`
}

var uuidRE = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

var validFieldTypes = map[string]struct{}{
	"string":   {},
	"number":   {},
	"integer":  {},
	"boolean":  {},
	"date":     {},
	"datetime": {},
}

var dataSourceCatalog = map[string]DataSourceDefinition{
	"Orders": {
		UUID: "220cc994-a913-4041-96fe-a96d7e0c97e8",
		Fields: map[string]FieldDefinition{
			"status": {
				FieldType: "string",
				Nullable:  false,
				SupportedOperators: toSet(
					"eq", "neq", "in", "nin", "contains", "startsWith", "endsWith", "exists", "isNull",
				),
				Description: "Current order status.",
			},
			"amount": {
				FieldType: "number",
				Nullable:  false,
				SupportedOperators: toSet(
					"eq", "neq", "gt", "gte", "lt", "lte", "in", "nin", "exists", "isNull",
				),
				Description: "Total order amount.",
			},
			"priority": {
				FieldType: "boolean",
				Nullable:  false,
				SupportedOperators: toSet(
					"eq", "neq", "exists", "isNull",
				),
				Description: "Whether the order is marked as priority.",
			},
			"customerEmail": {
				FieldType: "string",
				Nullable:  true,
				SupportedOperators: toSet(
					"eq", "neq", "contains", "startsWith", "endsWith", "exists", "isNull",
				),
				Description: "Customer email address.",
			},
			"createdAt": {
				FieldType: "datetime",
				Nullable:  false,
				SupportedOperators: toSet(
					"eq", "neq", "gt", "gte", "lt", "lte", "in", "nin", "exists", "isNull",
				),
				Description: "Order creation timestamp.",
			},
		},
	},
}

type DataSourceDefinition struct {
	UUID   string
	Fields map[string]FieldDefinition
}

// toSet creates a compact lookup set used for operator validation.
func toSet(values ...string) map[string]struct{} {
	m := make(map[string]struct{}, len(values))
	for _, v := range values {
		m[v] = struct{}{}
	}
	return m
}

// ValidateRequest performs envelope and datasource consistency checks.
func ValidateRequest(req FilterRequest) error {
	if err := validateRequestSchemaVersion(req.SchemaVersion); err != nil {
		return err
	}
	if !isUUID(req.RequestUUID) {
		return fmt.Errorf("invalid RequestUuid: %q", req.RequestUUID)
	}
	if !isUUID(req.DataSourceUUID) {
		return fmt.Errorf("invalid DataSourceUuid: %q", req.DataSourceUUID)
	}
	if req.DataSourceName == "" {
		return errors.New("DataSourceName is required")
	}
	if len(req.RequestFilter) == 0 {
		return errors.New("RequestFilter is required")
	}

	ds, ok := dataSourceCatalog[req.DataSourceName]
	if !ok {
		return fmt.Errorf("unknown DataSourceName: %q", req.DataSourceName)
	}
	if !strings.EqualFold(ds.UUID, req.DataSourceUUID) {
		return fmt.Errorf("DataSourceName/DataSourceUuid mismatch for %q", req.DataSourceName)
	}
	return nil
}

// GetAllowedFieldsResponse returns sorted field metadata for a known datasource.
func GetAllowedFieldsResponse(schemaVersion, requestUUID, dataSourceUUID, dataSourceName string) (AllowedFieldResponse, error) {
	if err := validateRequestSchemaVersion(schemaVersion); err != nil {
		return AllowedFieldResponse{}, err
	}
	if !isUUID(requestUUID) {
		return AllowedFieldResponse{}, fmt.Errorf("invalid RequestUuid: %q", requestUUID)
	}
	if !isUUID(dataSourceUUID) {
		return AllowedFieldResponse{}, fmt.Errorf("invalid DataSourceUuid: %q", dataSourceUUID)
	}
	ds, ok := dataSourceCatalog[dataSourceName]
	if !ok {
		return AllowedFieldResponse{}, fmt.Errorf("unknown DataSourceName: %q", dataSourceName)
	}
	if !strings.EqualFold(ds.UUID, dataSourceUUID) {
		return AllowedFieldResponse{}, fmt.Errorf("DataSourceName/DataSourceUuid mismatch for %q", dataSourceName)
	}

	items := make([]AllowedFieldResult, 0, len(ds.Fields))
	for fieldName, def := range ds.Fields {
		ops := sortedOperators(def.SupportedOperators)
		items = append(items, AllowedFieldResult{
			FieldName:          fieldName,
			FieldType:          def.FieldType,
			Nullable:           def.Nullable,
			SupportedOperators: ops,
			Description:        def.Description,
		})
	}

	sortAllowedFields(items)

	return AllowedFieldResponse{
		SchemaVersion:  schemaVersion,
		RequestUUID:    requestUUID,
		DataSourceUUID: dataSourceUUID,
		DataSourceName: dataSourceName,
		AllowedFields:  items,
	}, nil
}

// CompileRequest validates request metadata and compiles RequestFilter to SQL.
func CompileRequest(req FilterRequest) (CompiledFilter, error) {
	if err := ValidateRequest(req); err != nil {
		return CompiledFilter{}, err
	}
	ds := dataSourceCatalog[req.DataSourceName]
	sql, args, err := compileExpression(req.RequestFilter, ds.Fields)
	if err != nil {
		return CompiledFilter{}, err
	}
	return CompiledFilter{
		WhereSQL: sql,
		Args:     args,
	}, nil
}

// compileExpression recursively compiles comparison/and/or/not nodes.
func compileExpression(raw json.RawMessage, fields map[string]FieldDefinition) (string, []interface{}, error) {
	var cmpProbe struct {
		Field *string `json:"field"`
		Op    *string `json:"op"`
	}
	if err := json.Unmarshal(raw, &cmpProbe); err == nil && cmpProbe.Field != nil && cmpProbe.Op != nil {
		var cmp Comparison
		if err := json.Unmarshal(raw, &cmp); err != nil {
			return "", nil, fmt.Errorf("invalid comparison: %w", err)
		}
		return compileComparison(cmp, fields)
	}

	var andProbe struct {
		And *[]json.RawMessage `json:"and"`
	}
	if err := json.Unmarshal(raw, &andProbe); err == nil && andProbe.And != nil {
		return compileLogical("AND", *andProbe.And, fields)
	}

	var orProbe struct {
		Or *[]json.RawMessage `json:"or"`
	}
	if err := json.Unmarshal(raw, &orProbe); err == nil && orProbe.Or != nil {
		return compileLogical("OR", *orProbe.Or, fields)
	}

	var notProbe struct {
		Not *json.RawMessage `json:"not"`
	}
	if err := json.Unmarshal(raw, &notProbe); err == nil && notProbe.Not != nil {
		sql, args, err := compileExpression(*notProbe.Not, fields)
		if err != nil {
			return "", nil, err
		}
		return fmt.Sprintf("(NOT %s)", sql), args, nil
	}

	return "", nil, errors.New("invalid expression: must be comparison, and, or, or not")
}

// compileLogical joins recursively compiled child expressions using AND/OR.
func compileLogical(op string, parts []json.RawMessage, fields map[string]FieldDefinition) (string, []interface{}, error) {
	if len(parts) == 0 {
		return "", nil, fmt.Errorf("%s expression must contain at least one item", strings.ToLower(op))
	}

	clauses := make([]string, 0, len(parts))
	var args []interface{}

	for _, part := range parts {
		sql, partArgs, err := compileExpression(part, fields)
		if err != nil {
			return "", nil, err
		}
		clauses = append(clauses, sql)
		args = append(args, partArgs...)
	}

	return "(" + strings.Join(clauses, " "+op+" ") + ")", args, nil
}

// compileComparison validates operator/value semantics and emits SQL + args.
func compileComparison(c Comparison, fields map[string]FieldDefinition) (string, []interface{}, error) {
	if c.Field == "" {
		return "", nil, errors.New("comparison.field is required")
	}
	if c.Op == "" {
		return "", nil, errors.New("comparison.op is required")
	}
	if !isSafeIdentifier(c.Field) {
		return "", nil, fmt.Errorf("unsafe field name: %s", c.Field)
	}

	fieldDef, ok := fields[c.Field]
	if !ok {
		return "", nil, fmt.Errorf("field %q is not allowed for this datasource", c.Field)
	}
	if _, ok := fieldDef.SupportedOperators[c.Op]; !ok {
		return "", nil, fmt.Errorf("operator %q is not allowed for field %q", c.Op, c.Field)
	}

	col := quoteIdentifier(c.Field)

	switch c.Op {
	case "eq":
		if err := validateScalarValue(c.Value, fieldDef.FieldType); err != nil {
			return "", nil, fmt.Errorf("field %q: %w", c.Field, err)
		}
		return fmt.Sprintf("(%s = ?)", col), []interface{}{c.Value}, nil

	case "neq":
		if err := validateScalarValue(c.Value, fieldDef.FieldType); err != nil {
			return "", nil, fmt.Errorf("field %q: %w", c.Field, err)
		}
		return fmt.Sprintf("(%s <> ?)", col), []interface{}{c.Value}, nil

	case "gt":
		if err := validateComparableValue(c.Value, fieldDef.FieldType); err != nil {
			return "", nil, fmt.Errorf("field %q: %w", c.Field, err)
		}
		return fmt.Sprintf("(%s > ?)", col), []interface{}{c.Value}, nil

	case "gte":
		if err := validateComparableValue(c.Value, fieldDef.FieldType); err != nil {
			return "", nil, fmt.Errorf("field %q: %w", c.Field, err)
		}
		return fmt.Sprintf("(%s >= ?)", col), []interface{}{c.Value}, nil

	case "lt":
		if err := validateComparableValue(c.Value, fieldDef.FieldType); err != nil {
			return "", nil, fmt.Errorf("field %q: %w", c.Field, err)
		}
		return fmt.Sprintf("(%s < ?)", col), []interface{}{c.Value}, nil

	case "lte":
		if err := validateComparableValue(c.Value, fieldDef.FieldType); err != nil {
			return "", nil, fmt.Errorf("field %q: %w", c.Field, err)
		}
		return fmt.Sprintf("(%s <= ?)", col), []interface{}{c.Value}, nil

	case "contains":
		s, ok := c.Value.(string)
		if !ok {
			return "", nil, errors.New("contains requires string value")
		}
		return fmt.Sprintf("(%s LIKE ?)", col), []interface{}{"%" + s + "%"}, nil

	case "startsWith":
		s, ok := c.Value.(string)
		if !ok {
			return "", nil, errors.New("startsWith requires string value")
		}
		return fmt.Sprintf("(%s LIKE ?)", col), []interface{}{s + "%"}, nil

	case "endsWith":
		s, ok := c.Value.(string)
		if !ok {
			return "", nil, errors.New("endsWith requires string value")
		}
		return fmt.Sprintf("(%s LIKE ?)", col), []interface{}{"%" + s}, nil

	case "exists":
		b, ok := c.Value.(bool)
		if !ok {
			return "", nil, errors.New("exists requires boolean value")
		}
		if b {
			return fmt.Sprintf("(%s IS NOT NULL)", col), nil, nil
		}
		return fmt.Sprintf("(%s IS NULL)", col), nil, nil

	case "isNull":
		b, ok := c.Value.(bool)
		if !ok {
			return "", nil, errors.New("isNull requires boolean value")
		}
		if b {
			return fmt.Sprintf("(%s IS NULL)", col), nil, nil
		}
		return fmt.Sprintf("(%s IS NOT NULL)", col), nil, nil

	case "in", "nin":
		items, err := normalizeArray(c.Value)
		if err != nil {
			return "", nil, fmt.Errorf("%s requires a non-empty array: %w", c.Op, err)
		}
		for _, item := range items {
			if err := validateScalarValue(item, fieldDef.FieldType); err != nil {
				return "", nil, fmt.Errorf("field %q: %w", c.Field, err)
			}
		}

		placeholders := make([]string, len(items))
		args := make([]interface{}, len(items))
		for i, v := range items {
			placeholders[i] = "?"
			args[i] = v
		}

		sqlOp := "IN"
		if c.Op == "nin" {
			sqlOp = "NOT IN"
		}
		return fmt.Sprintf("(%s %s (%s))", col, sqlOp, strings.Join(placeholders, ", ")), args, nil
	}

	return "", nil, fmt.Errorf("operator not implemented: %s", c.Op)
}

func validateScalarValue(v interface{}, fieldType string) error {
	switch fieldType {
	case "string", "date", "datetime":
		if _, ok := v.(string); !ok {
			return fmt.Errorf("expected string value for field type %q", fieldType)
		}
		return nil
	case "number":
		if !isJSONNumber(v) {
			return fmt.Errorf("expected numeric value for field type %q", fieldType)
		}
		return nil
	case "integer":
		if !isJSONInteger(v) {
			return fmt.Errorf("expected integer value for field type %q", fieldType)
		}
		return nil
	case "boolean":
		if _, ok := v.(bool); !ok {
			return fmt.Errorf("expected boolean value for field type %q", fieldType)
		}
		return nil
	default:
		return fmt.Errorf("unsupported field type %q", fieldType)
	}
}

func validateComparableValue(v interface{}, fieldType string) error {
	switch fieldType {
	case "number", "integer", "date", "datetime":
		return validateScalarValue(v, fieldType)
	default:
		return fmt.Errorf("operator requires comparable field type, got %q", fieldType)
	}
}

func normalizeArray(v interface{}) ([]interface{}, error) {
	items, ok := v.([]interface{})
	if !ok || len(items) == 0 {
		return nil, errors.New("value is not a non-empty array")
	}
	return items, nil
}

// isSafeIdentifier prevents SQL injection through dynamic field names.
func isSafeIdentifier(s string) bool {
	if s == "" {
		return false
	}
	for i, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || r == '_' || (i > 0 && r >= '0' && r <= '9') {
			continue
		}
		return false
	}
	return true
}

func quoteIdentifier(s string) string {
	return `"` + s + `"`
}

func isUUID(s string) bool {
	return uuidRE.MatchString(s)
}

func isJSONNumber(v interface{}) bool {
	switch v.(type) {
	case float64:
		return true
	default:
		return false
	}
}

func isJSONInteger(v interface{}) bool {
	f, ok := v.(float64)
	if !ok {
		return false
	}
	return f == float64(int64(f))
}

func sortedOperators(m map[string]struct{}) []string {
	order := []string{
		"eq", "neq", "gt", "gte", "lt", "lte",
		"in", "nin", "contains", "startsWith", "endsWith",
		"exists", "isNull",
	}
	out := make([]string, 0, len(m))
	for _, op := range order {
		if _, ok := m[op]; ok {
			out = append(out, op)
		}
	}
	return out
}

func sortAllowedFields(items []AllowedFieldResult) {
	for i := 0; i < len(items)-1; i++ {
		for j := i + 1; j < len(items); j++ {
			if items[j].FieldName < items[i].FieldName {
				items[i], items[j] = items[j], items[i]
			}
		}
	}
}
