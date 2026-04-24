<script setup lang="ts">
import type { DataSourceListItem } from '../../types/api'

defineProps<{
  items: DataSourceListItem[]
}>()
</script>

<template>
  <div class="stack">
    <article v-for="item in items" :key="item.id" class="field-card stack">
      <header>
        <div>
          <h3>{{ item.label }}</h3>
          <p class="muted">{{ item.dataSourceName }} · {{ item.dataSourceUuid }}</p>
        </div>
      </header>
      <div class="cluster">
        <router-link
          v-for="source in item.supportedSources"
          :key="source"
          class="button secondary"
          :to="`/builder/${item.id}?source=${source}`"
        >
          Open {{ source }}
        </router-link>
      </div>
    </article>
  </div>
</template>
