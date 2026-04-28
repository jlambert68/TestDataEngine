import type { BuilderState, ComparisonExpression, FilterExpression, FilterGroupState, FilterRuleState } from '../types/api'

export function buildRequestFilter(state: BuilderState): FilterExpression | null {
  return buildGroupExpression(state.rootGroup, true)
}

function buildGroupExpression(group: FilterGroupState, isRoot = false): FilterExpression | null {
  const clauses = group.items
    .map(item => item.kind === 'rule' ? buildRuleExpression(item.rule) : buildGroupExpression(item.group))
    .filter((item): item is FilterExpression => item !== null)

  if (!clauses.length) {
    return null
  }
  let expression: FilterExpression
  if (isRoot && clauses.length === 1) {
    expression = clauses[0]
  } else {
    expression = group.combinator === 'or' ? { or: clauses } : { and: clauses }
  }

  return group.negated ? { not: expression } : expression
}

function buildRuleExpression(rule: FilterRuleState): FilterExpression | null {
  if (!rule.field) {
    return null
  }

  switch (rule.operator) {
    case 'eq':
    case 'neq':
      if (rule.values.length > 0) {
        return { field: rule.field, op: rule.operator, value: rule.values[0] }
      }
      if (rule.scalarValue === '' || rule.scalarValue === null) {
        return null
      }
      return { field: rule.field, op: rule.operator, value: rule.scalarValue }
    case 'in':
    case 'nin':
      if (!rule.values.length) {
        return null
      }
      return { field: rule.field, op: rule.operator, value: rule.values }
    case 'exists':
    case 'isNull':
      return { field: rule.field, op: rule.operator, value: rule.booleanValue }
    default:
      if (rule.values.length > 0) {
        return { field: rule.field, op: rule.operator, value: rule.values[0] }
      }
      if (rule.scalarValue === '' || rule.scalarValue === null) {
        return null
      }
      return { field: rule.field, op: rule.operator, value: rule.scalarValue }
  }
}
