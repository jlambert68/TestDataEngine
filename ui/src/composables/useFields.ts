import { ref } from 'vue'

import type { FieldDescriptor, GetFieldsResponse, SourceType } from '../types/api'

export function useFields() {
  const fields = ref<FieldDescriptor[]>([])
  const loading = ref(false)
  const error = ref('')

  async function load(datasourceId: string, source: SourceType) {
    loading.value = true
    error.value = ''
    try {
      const resp = await fetch(`/api/v1/datasources/${datasourceId}/fields?source=${source}`)
      if (!resp.ok) {
        throw new Error(`fields request failed: ${resp.status}`)
      }
      const body = (await resp.json()) as GetFieldsResponse
      fields.value = body.fields
    } catch (err) {
      error.value = err instanceof Error ? err.message : 'unknown error'
    } finally {
      loading.value = false
    }
  }

  return { fields, loading, error, load }
}
