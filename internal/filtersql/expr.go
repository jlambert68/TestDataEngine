package filtersql

import (
	"encoding/json"
	"errors"
	"fmt"
)

type Operator string

const (
	OpEq         Operator = "eq"
	OpNeq        Operator = "neq"
	OpGt         Operator = "gt"
	OpGte        Operator = "gte"
	OpLt         Operator = "lt"
	OpLte        Operator = "lte"
	OpIn         Operator = "in"
	OpNin        Operator = "nin"
	OpContains   Operator = "contains"
	OpStartsWith Operator = "startsWith"
	OpEndsWith   Operator = "endsWith"
	OpExists     Operator = "exists"
)

func (op Operator) Valid() bool {
	switch op {
	case OpEq, OpNeq, OpGt, OpGte, OpLt, OpLte, OpIn, OpNin, OpContains, OpStartsWith, OpEndsWith, OpExists:
		return true
	default:
		return false
	}
}

type Expression interface {
	isExpression()
}

type Scalar = any

type Comparison struct {
	Field string   `json:"field"`
	Op    Operator `json:"op"`
	Value any      `json:"value,omitempty"`
}

func (Comparison) isExpression() {}

type AndExpression struct {
	And []Expression `json:"and"`
}

func (AndExpression) isExpression() {}

type OrExpression struct {
	Or []Expression `json:"or"`
}

func (OrExpression) isExpression() {}

type NotExpression struct {
	Not Expression `json:"not"`
}

func (NotExpression) isExpression() {}

func ParseExpression(data []byte) (Expression, error) {
	var probe map[string]json.RawMessage
	if err := json.Unmarshal(data, &probe); err != nil {
		return nil, fmt.Errorf("expression must be an object: %w", err)
	}

	switch {
	case hasOnlyKey(probe, "and"):
		var raw struct {
			And []json.RawMessage `json:"and"`
		}
		if err := json.Unmarshal(data, &raw); err != nil {
			return nil, err
		}
		out := AndExpression{And: make([]Expression, 0, len(raw.And))}
		for i, item := range raw.And {
			expr, err := ParseExpression(item)
			if err != nil {
				return nil, fmt.Errorf("and[%d]: %w", i, err)
			}
			out.And = append(out.And, expr)
		}
		return out, nil

	case hasOnlyKey(probe, "or"):
		var raw struct {
			Or []json.RawMessage `json:"or"`
		}
		if err := json.Unmarshal(data, &raw); err != nil {
			return nil, err
		}
		out := OrExpression{Or: make([]Expression, 0, len(raw.Or))}
		for i, item := range raw.Or {
			expr, err := ParseExpression(item)
			if err != nil {
				return nil, fmt.Errorf("or[%d]: %w", i, err)
			}
			out.Or = append(out.Or, expr)
		}
		return out, nil

	case hasOnlyKey(probe, "not"):
		var raw struct {
			Not json.RawMessage `json:"not"`
		}
		if err := json.Unmarshal(data, &raw); err != nil {
			return nil, err
		}
		expr, err := ParseExpression(raw.Not)
		if err != nil {
			return nil, fmt.Errorf("not: %w", err)
		}
		return NotExpression{Not: expr}, nil

	default:
		var cmp Comparison
		if err := json.Unmarshal(data, &cmp); err != nil {
			return nil, err
		}
		if cmp.Field == "" && cmp.Op == "" {
			return nil, errors.New("expression must be one of comparison / and / or / not")
		}
		return cmp, nil
	}
}

func hasOnlyKey(m map[string]json.RawMessage, key string) bool {
	if len(m) != 1 {
		return false
	}
	_, ok := m[key]
	return ok
}
