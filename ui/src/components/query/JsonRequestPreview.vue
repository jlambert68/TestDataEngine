<script setup lang="ts">
import { computed, ref } from 'vue'

import type { QueryPreviewRequest } from '../../types/api'

const props = defineProps<{
  request: QueryPreviewRequest | null
  compareStatus: 'idle' | 'same' | 'different' | 'error'
  compareMessage: string
}>()

const emit = defineEmits<{
  'apply-payload': [payload: string]
}>()

const importPayload = ref('')
const rendered = computed(() => JSON.stringify(props.request, null, 2))

function applyPayload() {
  emit('apply-payload', importPayload.value)
}
</script>

<template>
  <article class="panel panel-pad stack">
    <p class="eyebrow">Outgoing Payload</p>
    <p v-if="!request" class="muted">Choose at least one field value to generate a filter request.</p>
    <pre v-else>{{ rendered }}</pre>

    <label class="stack">
      <span class="muted">Paste payload to rebuild Filter Builder</span>
      <textarea
        v-model="importPayload"
        rows="10"
        placeholder="Paste an Outgoing Payload JSON object here"
      />
    </label>
    <div class="cluster">
      <button class="button secondary" type="button" @click="applyPayload">Apply Payload</button>
      <span
        v-if="compareStatus !== 'idle'"
        class="chip"
        :class="{
          'payload-status-same': compareStatus === 'same',
          'payload-status-different': compareStatus === 'different',
          'payload-status-error': compareStatus === 'error',
        }"
      >
        {{
          compareStatus === 'same'
            ? 'Payload matches'
            : compareStatus === 'different'
              ? 'Payload differs'
              : 'Payload import failed'
        }}
      </span>
    </div>
    <p v-if="compareMessage" class="muted">{{ compareMessage }}</p>
  </article>
</template>
