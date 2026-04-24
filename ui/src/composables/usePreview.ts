import { ref } from 'vue'

import type { QueryPreviewRequest, QueryPreviewResponse } from '../types/api'

export function usePreview() {
  const loading = ref(false)
  const error = ref('')
  const response = ref<QueryPreviewResponse | null>(null)

  async function run(payload: QueryPreviewRequest) {
    loading.value = true
    error.value = ''
    try {
      const resp = await fetch('/api/v1/query/preview', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify(payload),
      })
      if (!resp.ok) {
        const body = await resp.text()
        throw new Error(body || `preview request failed: ${resp.status}`)
      }
      response.value = (await resp.json()) as QueryPreviewResponse
    } catch (err) {
      error.value = err instanceof Error ? err.message : 'unknown error'
      response.value = null
    } finally {
      loading.value = false
    }
  }

  return { loading, error, response, run }
}
