<script setup lang="ts">
import { computed, ref } from 'vue'

import { useFacets } from '../../composables/useFacets'
import type { Scalar, SourceType } from '../../types/api'

const props = defineProps<{
  datasourceId: string
  source: SourceType
  fieldName: string
  nullable: boolean
  modelValue: Scalar[]
  allowMultiple: boolean
  allowNullValue: boolean
}>()

const emit = defineEmits<{
  'update:modelValue': [value: Scalar[]]
}>()

const search = ref('')
const { items, loading, error, truncated } = useFacets(
  () => props.datasourceId,
  () => props.source,
  () => props.fieldName,
  () => search.value,
)

const visibleItems = computed(() =>
  props.allowNullValue ? items.value : items.value.filter(item => !item.isNull),
)

function toggleValue(value: Scalar, checked: boolean) {
  if (!props.allowMultiple) {
    emit('update:modelValue', checked ? [value] : [])
    return
  }

  const next = new Set(props.modelValue)
  if (checked) {
    next.add(value)
  } else {
    next.delete(value)
  }
  emit('update:modelValue', Array.from(next))
}

function isChecked(value: Scalar) {
  return props.modelValue.some(item => item === value)
}

function onValueChange(value: Scalar, event: Event) {
  toggleValue(value, (event.target as HTMLInputElement).checked)
}
</script>

<template>
  <div class="stack">
    <label class="stack">
      <span class="muted">Search values</span>
      <input v-model="search" type="search" placeholder="Filter checkbox values…" />
    </label>

    <div class="facet-list">
      <p v-if="loading" class="muted">Loading values…</p>
      <p v-else-if="error" class="muted">{{ error }}</p>
      <template v-else>
        <label v-for="item in visibleItems" :key="`${fieldName}-${item.label}`" class="facet-row">
          <input
            :checked="isChecked(item.value)"
            :type="allowMultiple ? 'checkbox' : 'radio'"
            :name="allowMultiple ? undefined : `facet-${fieldName}`"
            @change="onValueChange(item.value, $event)"
          />
          <span>{{ item.label }}</span>
          <span class="muted">{{ item.count }}</span>
        </label>
        <p v-if="truncated" class="muted">Showing the first 100 facet values.</p>
      </template>
    </div>
  </div>
</template>
