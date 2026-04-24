import { ref, watch } from 'vue'

import type { FacetValue, GetFacetsResponse, SourceType } from '../types/api'

export function useFacets(datasourceId: () => string, source: () => SourceType, field: () => string, search: () => string) {
  const items = ref<FacetValue[]>([])
  const loading = ref(false)
  const error = ref('')
  const truncated = ref(false)

  async function load() {
    const ds = datasourceId()
    const src = source()
    const fld = field()
    if (!ds || !src || !fld) {
      items.value = []
      return
    }

    loading.value = true
    error.value = ''
    try {
      const params = new URLSearchParams({
        source: src,
        field: fld,
        limit: '100',
      })
      if (search()) {
        params.set('q', search())
      }
      const resp = await fetch(`/api/v1/datasources/${ds}/facets?${params.toString()}`)
      if (!resp.ok) {
        throw new Error(`facets request failed: ${resp.status}`)
      }
      const body = (await resp.json()) as GetFacetsResponse
      items.value = body.values
      truncated.value = body.truncated
    } catch (err) {
      error.value = err instanceof Error ? err.message : 'unknown error'
    } finally {
      loading.value = false
    }
  }

  watch([datasourceId, source, field, search], () => {
    void load()
  }, { immediate: true })

  return { items, loading, error, truncated, reload: load }
}
