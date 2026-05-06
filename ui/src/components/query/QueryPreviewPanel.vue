<script setup lang="ts">
import { computed } from 'vue'

import { usePreview } from '../../composables/usePreview'
import type { QueryPreviewRequest } from '../../types/api'

const props = defineProps<{
  request: QueryPreviewRequest | null
  maxItems: number
  randomSeedGuid: string
  randomSeedOffset: number
}>()

const emit = defineEmits<{
  'update:maxItems': [value: number]
  'update:random-seed-guid': [value: string]
  'update:random-seed-offset': [value: number]
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

function onRandomSeedGuidInput(event: Event) {
  emit('update:random-seed-guid', (event.target as HTMLInputElement).value)
}

function onRandomSeedOffsetInput(event: Event) {
  const raw = Number.parseInt((event.target as HTMLInputElement).value, 10)
  const next = Number.isNaN(raw) ? 0 : Math.max(0, raw)
  emit('update:random-seed-offset', next)
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
      <label class="stack" style="gap: 6px; min-width: 220px;">
        <span class="muted">Random seed GUID</span>
        <input
          :value="randomSeedGuid"
          type="text"
          placeholder="Optional UUID"
          @input="onRandomSeedGuidInput"
        />
      </label>
      <label class="stack" style="gap: 6px; min-width: 160px;">
        <span class="muted">Seed offset</span>
        <input
          :value="randomSeedOffset"
          type="number"
          min="0"
          step="1"
          @input="onRandomSeedOffsetInput"
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
