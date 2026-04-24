<script setup lang="ts">
import { computed } from 'vue'

import { usePreview } from '../../composables/usePreview'
import type { QueryPreviewRequest } from '../../types/api'

const props = defineProps<{
  request: QueryPreviewRequest | null
  maxItems: number
}>()

const emit = defineEmits<{
  'update:maxItems': [value: number]
}>()

const { loading, error, response, run } = usePreview()

const rows = computed(() => response.value?.dataSet.TestData?.TestDataSet ?? [])

function preview() {
  if (!props.request) {
    return
  }
  void run(props.request)
}

function onMaxItemsInput(event: Event) {
  const raw = Number.parseInt((event.target as HTMLInputElement).value, 10)
  const next = Number.isNaN(raw) ? 1 : Math.min(100, Math.max(1, raw))
  emit('update:maxItems', next)
}
</script>

<template>
  <article class="panel panel-pad stack">
    <div class="cluster">
      <p class="eyebrow">Preview</p>
      <label class="stack" style="gap: 6px; min-width: 160px;">
        <span class="muted">Rows to show</span>
        <input
          :value="maxItems"
          type="number"
          min="1"
          max="100"
          @input="onMaxItemsInput"
        />
      </label>
      <button class="button" type="button" :disabled="!request || loading" @click="preview">
        {{ loading ? 'Running…' : 'Run Preview' }}
      </button>
    </div>
    <p class="muted">Preview returns at most 100 rows.</p>

    <p v-if="error" class="muted">{{ error }}</p>
    <template v-else-if="response">
      <div class="stack">
        <div class="chip">WHERE {{ response.compiledWhereSql }}</div>
        <pre>{{ JSON.stringify(response.compiledArgs, null, 2) }}</pre>
      </div>

      <table v-if="rows.length" class="results-table">
        <thead>
          <tr>
            <th>Row</th>
            <th>Payload</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="(row, index) in rows" :key="index">
            <td>{{ index + 1 }}</td>
            <td><pre>{{ JSON.stringify(row, null, 2) }}</pre></td>
          </tr>
        </tbody>
      </table>
      <p v-else class="muted">No rows matched.</p>
    </template>
    <p v-else class="muted">Preview results will appear here.</p>
  </article>
</template>
