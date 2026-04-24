<script setup lang="ts">
import { computed } from 'vue'

import type { QueryPreviewRequest } from '../../types/api'

const props = defineProps<{
  request: QueryPreviewRequest | null
}>()

const rendered = computed(() => JSON.stringify(props.request, null, 2))
</script>

<template>
  <article class="panel panel-pad stack">
    <p class="eyebrow">Outgoing Payload</p>
    <p v-if="!request" class="muted">Choose at least one field value to generate a filter request.</p>
    <pre v-else>{{ rendered }}</pre>
  </article>
</template>
