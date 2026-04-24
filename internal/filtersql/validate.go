package filtersql

import (
	"errors"
	"fmt"
	"regexp"

	"TestDataEngine/internal/filters"
)

var uuidRE = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[1-5][0-9a-fA-F]{3}-[89abAB][0-9a-fA-F]{3}-[0-9a-fA-F]{12}$`)

// ValidateRequest validates the top-level request envelope and nested filter tree.
func ValidateRequest(r Request) error {
	expected, err := filters.RequestSchemaVersion()
	if err != nil {
		return fmt.Errorf("load request schema version: %w", err)
	}
	if r.SchemaVersion != expected {
		return fmt.Errorf("SchemaVersion must be %q", expected)
	}
	if !uuidRE.MatchString(r.RequestUUID) {
		return errors.New("RequestUuid must be a valid UUID")
	}
	if !uuidRE.MatchString(r.DataSourceUUID) {
		return errors.New("DataSourceUuid must be a valid UUID")
	}
	if r.DataSourceName == "" {
		return errors.New("DataSourceName is required")
	}
	if r.RequestFilter == nil {
		return errors.New("RequestFilter is required")
	}
	return ValidateExpression(r.RequestFilter)
}

// ValidateExpression validates the recursive shape and values of an expression tree.
func ValidateExpression(expr Expression) error {
	switch e := expr.(type) {
	case Comparison:
		return validateComparison(e)
	case AndExpression:
		if len(e.And) == 0 {
			return errors.New(`"and" must contain at least one expression`)
		}
		for i, child := range e.And {
			if err := ValidateExpression(child); err != nil {
				return fmt.Errorf("and[%d]: %w", i, err)
			}
		}
		return nil
	case OrExpression:
		if len(e.Or) == 0 {
			return errors.New(`"or" must contain at least one expression`)
		}
		for i, child := range e.Or {
			if err := ValidateExpression(child); err != nil {
				return fmt.Errorf("or[%d]: %w", i, err)
			}
		}
		return nil
	case NotExpression:
		if e.Not == nil {
			return errors.New(`"not" must contain an expression`)
		}
		return ValidateExpression(e.Not)
	default:
		return fmt.Errorf("unsupported expression type %T", expr)
	}
}

// validateComparison validates one comparison clause for operator/value compatibility.
func validateComparison(c Comparison) error {
	if c.Field == "" {
		return errors.New("comparison.field is required")
	}
	if !c.Op.Valid() {
		return fmt.Errorf("unsupported operator %q", c.Op)
	}

	switch c.Op {
	case OpExists, OpIsNull:
		v, ok := c.Value.(bool)
		if !ok {
			return fmt.Errorf(`operator %q requires boolean value`, c.Op)
		}
		_ = v
		return nil

	case OpIn, OpNin:
		arr, ok := c.Value.([]any)
		if !ok || len(arr) == 0 {
			return fmt.Errorf(`operator %q requires non-empty array value`, c.Op)
		}
		for i, item := range arr {
			if !isScalar(item) {
				return fmt.Errorf("%q value[%d] must be scalar", c.Op, i)
			}
		}
		return nil

	case OpContains, OpStartsWith, OpEndsWith:
		if _, ok := c.Value.(string); !ok {
			return fmt.Errorf(`operator %q requires string value`, c.Op)
		}
		return nil

	default:
		if c.Value == nil {
			return fmt.Errorf(`operator %q requires value`, c.Op)
		}
		if !isScalar(c.Value) {
			return fmt.Errorf(`operator %q requires scalar value`, c.Op)
		}
		return nil
	}
}

// isScalar reports whether v can be represented as a scalar filter value.
func isScalar(v any) bool {
	switch v.(type) {
	case nil, string, bool, float64, int, int8, int16, int32, int64, float32, uint, uint8, uint16, uint32, uint64:
		return true
	default:
		return false
	}
}
