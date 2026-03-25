package filtersql

import (
	"fmt"
	"regexp"
	"strings"
)

type PlaceholderStyle int

const (
	Question PlaceholderStyle = iota // ?, ?, ?
	Dollar                           // $1, $2, $3
)

type Compiler struct {
	Placeholder PlaceholderStyle
	QuoteIdent  bool
}

func (c Compiler) Compile(req Request) (string, []any, error) {
	if err := ValidateRequest(req); err != nil {
		return "", nil, err
	}
	sql, args, err := c.compileExpr(req.RequestFilter, 1)
	if err != nil {
		return "", nil, err
	}
	return sql, args, nil
}

func (c Compiler) compileExpr(expr Expression, next int) (string, []any, error) {
	switch e := expr.(type) {
	case Comparison:
		return c.compileComparison(e, next)

	case AndExpression:
		var parts []string
		var args []any
		for _, child := range e.And {
			part, childArgs, err := c.compileExpr(child, next+len(args))
			if err != nil {
				return "", nil, err
			}
			parts = append(parts, "("+part+")")
			args = append(args, childArgs...)
		}
		return strings.Join(parts, " AND "), args, nil

	case OrExpression:
		var parts []string
		var args []any
		for _, child := range e.Or {
			part, childArgs, err := c.compileExpr(child, next+len(args))
			if err != nil {
				return "", nil, err
			}
			parts = append(parts, "("+part+")")
			args = append(args, childArgs...)
		}
		return strings.Join(parts, " OR "), args, nil

	case NotExpression:
		part, args, err := c.compileExpr(e.Not, next)
		if err != nil {
			return "", nil, err
		}
		return "NOT (" + part + ")", args, nil

	default:
		return "", nil, fmt.Errorf("unsupported expression type %T", expr)
	}
}

func (c Compiler) compileComparison(cmp Comparison, next int) (string, []any, error) {
	field, err := c.identifier(cmp.Field)
	if err != nil {
		return "", nil, err
	}

	switch cmp.Op {
	case OpEq:
		if cmp.Value == nil {
			return field + " IS NULL", nil, nil
		}
		return fmt.Sprintf("%s = %s", field, c.ph(next)), []any{cmp.Value}, nil

	case OpNeq:
		if cmp.Value == nil {
			return field + " IS NOT NULL", nil, nil
		}
		return fmt.Sprintf("%s <> %s", field, c.ph(next)), []any{cmp.Value}, nil

	case OpGt:
		return fmt.Sprintf("%s > %s", field, c.ph(next)), []any{cmp.Value}, nil
	case OpGte:
		return fmt.Sprintf("%s >= %s", field, c.ph(next)), []any{cmp.Value}, nil
	case OpLt:
		return fmt.Sprintf("%s < %s", field, c.ph(next)), []any{cmp.Value}, nil
	case OpLte:
		return fmt.Sprintf("%s <= %s", field, c.ph(next)), []any{cmp.Value}, nil

	case OpIn, OpNin:
		items := cmp.Value.([]any)
		holders := make([]string, 0, len(items))
		args := make([]any, 0, len(items))
		for i, v := range items {
			holders = append(holders, c.ph(next+i))
			args = append(args, v)
		}
		kw := "IN"
		if cmp.Op == OpNin {
			kw = "NOT IN"
		}
		return fmt.Sprintf("%s %s (%s)", field, kw, strings.Join(holders, ", ")), args, nil

	case OpContains:
		return fmt.Sprintf("%s LIKE %s", field, c.ph(next)), []any{"%" + cmp.Value.(string) + "%"}, nil

	case OpStartsWith:
		return fmt.Sprintf("%s LIKE %s", field, c.ph(next)), []any{cmp.Value.(string) + "%"}, nil

	case OpEndsWith:
		return fmt.Sprintf("%s LIKE %s", field, c.ph(next)), []any{"%" + cmp.Value.(string)}, nil

	case OpExists:
		exists := cmp.Value.(bool)
		if exists {
			return field + " IS NOT NULL", nil, nil
		}
		return field + " IS NULL", nil, nil

	default:
		return "", nil, fmt.Errorf("unsupported operator %q", cmp.Op)
	}
}

var identRE = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

func (c Compiler) identifier(name string) (string, error) {
	if !identRE.MatchString(name) {
		return "", fmt.Errorf("invalid field name %q", name)
	}
	if !c.QuoteIdent {
		return name, nil
	}
	return `"` + name + `"`, nil
}

func (c Compiler) ph(n int) string {
	switch c.Placeholder {
	case Dollar:
		return fmt.Sprintf("$%d", n)
	default:
		return "?"
	}
}
