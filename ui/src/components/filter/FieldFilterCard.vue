<script setup lang="ts">
import { computed } from 'vue'

import FacetCheckboxList from './FacetCheckboxList.vue'
import type { ComparisonExpression, FieldDescriptor, FilterRuleState, Scalar, SourceType } from '../../types/api'

const props = defineProps<{
  datasourceId: string
  source: SourceType
  fields: FieldDescriptor[]
  rule: FilterRuleState
  removable?: boolean
}>()

const emit = defineEmits<{
  remove: []
}>()

const selectedField = computed(() =>
  props.fields.find(item => item.field === props.rule.field),
)

const operatorOptions = computed(() => selectedField.value?.supportedOperators ?? [])

const usesBooleanToggle = computed(() =>
  ['exists', 'isNull'].includes(props.rule.operator),
)

const usesFacetSelection = computed(() =>
  !!selectedField.value?.facetEligible &&
  !usesBooleanToggle.value &&
  !['startsWith', 'endsWith'].includes(props.rule.operator),
)

const usesTextInput = computed(() => !usesFacetSelection.value && !usesBooleanToggle.value)

const allowsMultipleFacetValues = computed(() =>
  ['in', 'nin'].includes(props.rule.operator),
)

const selectedCount = computed(() => props.rule.values.length)

function updateField(field: string) {
  props.rule.field = field
  props.rule.values = []
  props.rule.scalarValue = ''
  props.rule.booleanValue = true
  const nextField = props.fields.find(item => item.field === field)
  props.rule.operator = defaultOperatorForField(nextField)
}

function onFieldChange(event: Event) {
  updateField((event.target as HTMLSelectElement).value)
}

function updateOperator(operator: ComparisonExpression['op']) {
  props.rule.operator = operator
  props.rule.values = []
  props.rule.scalarValue = ''
  props.rule.booleanValue = true
}

function onOperatorChange(event: Event) {
  updateOperator((event.target as HTMLSelectElement).value as ComparisonExpression['op'])
}

function updateValues(values: Scalar[]) {
  props.rule.values = values
  if (!allowsMultipleFacetValues.value) {
    props.rule.scalarValue = values[0] ?? ''
  }
}

function updateScalarValue(value: Scalar) {
  props.rule.scalarValue = value
}

function onScalarInput(event: Event) {
  if (!selectedField.value) {
    return
  }
  const raw = (event.target as HTMLInputElement).value
  switch (selectedField.value.fieldType) {
    case 'integer':
      updateScalarValue(raw === '' ? '' : Number.parseInt(raw, 10))
      return
    case 'number':
      updateScalarValue(raw === '' ? '' : Number.parseFloat(raw))
      return
    default:
      updateScalarValue(raw)
  }
}

function updateBooleanValue(value: boolean) {
  props.rule.booleanValue = value
}

function clear() {
  props.rule.values = []
  props.rule.scalarValue = ''
  props.rule.booleanValue = true
}

function defaultOperatorForField(field?: FieldDescriptor) {
  if (!field) {
    return 'eq'
  }
  if (field.supportedOperators.includes('eq')) {
    return 'eq'
  }
  return (field.supportedOperators[0] ?? 'eq') as ComparisonExpression['op']
}

function inputTypeForField(field?: FieldDescriptor) {
  switch (field?.fieldType) {
    case 'integer':
    case 'number':
      return 'number'
    default:
      return 'text'
  }
}

function operatorHelp(operator: ComparisonExpression['op']) {
  switch (operator) {
    case 'eq':
      return 'Exact match'
    case 'neq':
      return 'Exclude one exact value'
    case 'in':
      return 'Match any selected values'
    case 'nin':
      return 'Exclude all selected values'
    case 'contains':
      return 'Substring match'
    case 'startsWith':
      return 'Prefix match'
    case 'endsWith':
      return 'Suffix match'
    case 'gt':
    case 'gte':
    case 'lt':
    case 'lte':
      return 'Ordered comparison'
    case 'exists':
      return 'Checks whether the value is present'
    case 'isNull':
      return 'Checks whether the value is null'
    default:
      return ''
  }
}
</script>

<template>
  <article class="field-card stack">
    <header>
      <div class="stack" style="gap: 10px;">
        <div class="rule-grid">
          <label class="stack">
            <span class="muted">Field</span>
            <select :value="rule.field" @change="onFieldChange">
              <option value="">Choose field…</option>
              <option v-for="field in fields" :key="field.field" :value="field.field">
                {{ field.field }}
              </option>
            </select>
          </label>

          <label class="stack">
            <span class="muted">Operator</span>
            <select :value="rule.operator" :disabled="!selectedField" @change="onOperatorChange">
              <option v-for="operator in operatorOptions" :key="operator" :value="operator">
                {{ operator }}
              </option>
            </select>
          </label>
        </div>
      </div>
      <div class="cluster">
        <button class="button secondary" type="button" @click="clear">Clear</button>
        <button v-if="removable" class="button secondary" type="button" @click="emit('remove')">Remove</button>
      </div>
    </header>

    <div class="cluster">
      <span v-if="usesFacetSelection" class="chip">{{ selectedCount }} selected</span>
      <span v-if="selectedField" class="chip">{{ selectedField.nullable ? 'Nullable' : 'Required' }}</span>
      <span v-if="selectedField" class="chip">{{ operatorHelp(rule.operator) }}</span>
    </div>

    <p v-if="selectedField?.description" class="muted">{{ selectedField.description }}</p>

    <FacetCheckboxList
      v-if="selectedField && usesFacetSelection"
      :datasource-id="datasourceId"
      :source="source"
      :field-name="selectedField.field"
      :nullable="selectedField.nullable"
      :model-value="rule.values"
      :allow-multiple="allowsMultipleFacetValues"
      :allow-null-value="false"
      @update:model-value="updateValues"
    />

    <div v-else-if="usesBooleanToggle" class="cluster">
      <label class="chip">
        <input :checked="rule.booleanValue === true" type="radio" :name="`bool-${rule.id}`" @change="updateBooleanValue(true)" />
        true
      </label>
      <label class="chip">
        <input :checked="rule.booleanValue === false" type="radio" :name="`bool-${rule.id}`" @change="updateBooleanValue(false)" />
        false
      </label>
    </div>

    <label v-else-if="selectedField && usesTextInput" class="stack">
      <span class="muted">Value</span>
      <input
        :type="inputTypeForField(selectedField)"
        :value="rule.scalarValue ?? ''"
        :placeholder="rule.operator"
        @input="onScalarInput"
      />
    </label>

    <p v-else class="muted">
      Select a field first. Then pick the operator you want to use for this rule.
    </p>
  </article>
</template>
