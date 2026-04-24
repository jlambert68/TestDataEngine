import { ref } from 'vue'

import type { DataSourceListItem, ListDataSourcesResponse } from '../types/api'

export function useDatasources() {
  const items = ref<DataSourceListItem[]>([])
  const loading = ref(false)
  const error = ref('')

  async function load() {
    loading.value = true
    error.value = ''
    try {
      const resp = await fetch('/api/v1/datasources')
      if (!resp.ok) {
        throw new Error(`datasources request failed: ${resp.status}`)
      }
      const body = (await resp.json()) as ListDataSourcesResponse
      items.value = body.items
    } catch (err) {
      error.value = err instanceof Error ? err.message : 'unknown error'
    } finally {
      loading.value = false
    }
  }

  return { items, loading, error, load }
}
