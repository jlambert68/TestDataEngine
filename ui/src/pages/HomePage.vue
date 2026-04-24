<script setup lang="ts">
import { computed, onMounted } from 'vue'
import { useRoute } from 'vue-router'

import DataSourcePicker from '../components/datasource/DataSourcePicker.vue'
import { useDatasources } from '../composables/useDatasources'

const route = useRoute()
const { items, loading, error, load } = useDatasources()
const returnTo = computed(() => String(route.query.returnTo ?? ''))

onMounted(() => {
  void load()
})
</script>

<template>
  <section class="page-grid home-grid">
    <article class="panel panel-pad stack">
      <p class="eyebrow">Start Here</p>
      <h2>Choose a datasource and source backend</h2>
      <p class="muted">
        The builder loads field metadata from the backend, then offers one filter card per field.
        Faceted fields become checkbox lists; null handling is expressed with explicit toggles.
      </p>
      <div v-if="returnTo" class="cluster">
        <router-link class="button secondary" :to="returnTo">Cancel / Ignore</router-link>
      </div>
      <p v-if="loading" class="muted">Loading datasources…</p>
      <p v-else-if="error" class="muted">{{ error }}</p>
      <DataSourcePicker v-else :items="items" />
    </article>

    <aside class="panel panel-pad stack">
      <p class="eyebrow">Current API</p>
      <h2>Backed by the Go filter contract</h2>
      <p class="muted">
        `GET /api/v1/datasources` lists datasets, `GET /api/v1/datasources/:id/fields`
        provides filterable columns, and `GET /api/v1/datasources/:id/facets` powers the checkbox lists.
      </p>
      <div class="chip">Preview endpoint: <code>/api/v1/query/preview</code></div>
    </aside>
  </section>
</template>
