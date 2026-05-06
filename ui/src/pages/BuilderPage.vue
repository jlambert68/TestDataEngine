<script setup lang="ts">
import { computed, onMounted, reactive, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'

import FilterGroupEditor from '../components/filter/FilterGroupEditor.vue'
import JsonRequestPreview from '../components/query/JsonRequestPreview.vue'
import QueryPreviewPanel from '../components/query/QueryPreviewPanel.vue'
import { useDatasources } from '../composables/useDatasources'
import { useFields } from '../composables/useFields'
import { buildRequestFilter } from '../utils/buildRequestFilter'
import { compareOutgoingPayloads, parseFilterGroupFromPayloadExpression, parseOutgoingPayloadInput } from '../utils/payloadRoundtrip'
import { makeFilterRequest } from '../utils/requestEnvelope'
import type { BuilderState, DataSourceListItem, FilterGroupState, FilterRuleState, QueryPreviewRequest, SourceType } from '../types/api'

const route = useRoute()
const router = useRouter()

const datasourceId = computed(() => String(route.params.datasourceId ?? ''))
const requestedSource = computed(() => String(route.query.source ?? ''))

const { items, load: loadDatasources } = useDatasources()
const { fields, loading, error, load: loadFields } = useFields()
const requestUuidOverride = ref('')
const payloadCompareStatus = ref<'idle' | 'same' | 'different' | 'error'>('idle')
const payloadCompareMessage = ref('')

const state = reactive<BuilderState>({
  datasourceId: '',
  source: 'sqlite',
  maxItems: 25,
  randomSeedGuid: '',
  randomSeedOffset: 0,
  rootGroup: createGroup('and'),
})

const activeDatasource = computed<DataSourceListItem | undefined>(() =>
  items.value.find(item => item.id === datasourceId.value),
)

const activeRequest = computed<QueryPreviewRequest | null>(() => {
  if (!activeDatasource.value) {
    return null
  }
  const expr = buildRequestFilter(state)
  if (!expr) {
    return null
  }
  return {
    source: state.source,
    maxItems: state.maxItems,
    randomSeedGuid: state.randomSeedGuid || undefined,
    randomSeedOffset: state.randomSeedOffset > 0 ? state.randomSeedOffset : undefined,
    request: makeFilterRequest(activeDatasource.value, expr, requestUuidOverride.value || undefined),
  }
})

watch([activeDatasource, requestedSource], ([datasource, source]) => {
  if (!datasource) {
    return
  }
  requestUuidOverride.value = ''
  resetPayloadCompare()
  state.datasourceId = datasource.id
  state.source = datasource.supportedSources.includes(source as SourceType)
    ? (source as SourceType)
    : datasource.defaultSource
  restoreBuilderState(datasource.id, state.source)
  void loadFields(datasource.id, state.source)
}, { immediate: true })

watch(state, () => {
  persistBuilderState()
}, { deep: true })

onMounted(async () => {
  await loadDatasources()
})

function createRule(field = ''): FilterRuleState {
  return {
    id: crypto.randomUUID(),
    field,
    operator: 'eq',
    values: [],
    scalarValue: '',
    booleanValue: true,
  }
}

function createGroup(combinator: 'and' | 'or'): FilterGroupState {
  return {
    id: crypto.randomUUID(),
    combinator,
    negated: false,
    items: [],
  }
}

function resetBuilder() {
  state.rootGroup = createGroup('and')
  requestUuidOverride.value = ''
  resetPayloadCompare()
}

function updateMaxItems(value: number) {
  state.maxItems = Math.min(100, Math.max(1, value))
}

function updateRandomSeedGuid(value: string) {
  state.randomSeedGuid = value
}

function updateRandomSeedOffset(value: number) {
  if (Number.isNaN(value)) {
    state.randomSeedOffset = 0
    return
  }
  state.randomSeedOffset = Math.max(0, Math.trunc(value))
}

function resetPayloadCompare() {
  payloadCompareStatus.value = 'idle'
  payloadCompareMessage.value = ''
}

async function applyOutgoingPayload(payload: string) {
  resetPayloadCompare()
  const datasource = activeDatasource.value
  if (!datasource) {
    payloadCompareStatus.value = 'error'
    payloadCompareMessage.value = 'Select a datasource before importing payload.'
    return
  }

  try {
    const parsed = parseOutgoingPayloadInput(payload)
    const targetDatasource = items.value.find(item =>
      item.dataSourceUuid === parsed.request.DataSourceUuid &&
      item.dataSourceName === parsed.request.DataSourceName,
    )
    if (!targetDatasource) {
      throw new Error('Payload datasource does not exist in the datasource catalog.')
    }
    if (!targetDatasource.supportedSources.includes(parsed.source)) {
      throw new Error(`Payload source "${parsed.source}" is not supported for this datasource.`)
    }

    if (targetDatasource.id !== datasource.id || parsed.source !== requestedSource.value) {
      await router.push({ path: `/builder/${targetDatasource.id}`, query: { source: parsed.source } })
    }

    state.datasourceId = targetDatasource.id
    state.source = parsed.source
    await loadFields(targetDatasource.id, parsed.source)
    if (error.value) {
      throw new Error(error.value)
    }

    state.maxItems = Math.min(100, Math.max(1, parsed.maxItems))
    state.randomSeedGuid = parsed.randomSeedGuid ?? ''
    state.randomSeedOffset = parsed.randomSeedOffset ?? 0
    state.rootGroup = parseFilterGroupFromPayloadExpression(parsed.request.RequestFilter, fields.value)
    requestUuidOverride.value = parsed.request.RequestUuid

    const rebuilt = activeRequest.value
    if (!rebuilt) {
      throw new Error('Could not rebuild outgoing payload from imported filter.')
    }

    const compare = compareOutgoingPayloads(parsed, rebuilt)
    payloadCompareStatus.value = compare.same ? 'same' : 'different'
    payloadCompareMessage.value = compare.same
      ? 'Imported payload matches the rebuilt payload.'
      : 'Imported payload does not match the rebuilt payload.'
  } catch (err) {
    payloadCompareStatus.value = 'error'
    payloadCompareMessage.value = err instanceof Error ? err.message : 'Failed to import payload.'
  }
}

function storageKey(datasource: string, source: SourceType) {
  return `testdataengine-builder:${datasource}:${source}`
}

function persistBuilderState() {
  if (!state.datasourceId) {
    return
  }
  sessionStorage.setItem(storageKey(state.datasourceId, state.source), JSON.stringify({
    maxItems: state.maxItems,
    randomSeedGuid: state.randomSeedGuid,
    randomSeedOffset: state.randomSeedOffset,
    rootGroup: state.rootGroup,
  }))
}

function restoreBuilderState(datasource: string, source: SourceType) {
  const raw = sessionStorage.getItem(storageKey(datasource, source))
  if (!raw) {
    state.maxItems = 25
    state.randomSeedGuid = ''
    state.randomSeedOffset = 0
    state.rootGroup = createGroup('and')
    return
  }

  try {
    const parsed = JSON.parse(raw) as Partial<BuilderState>
    state.maxItems = typeof parsed.maxItems === 'number' ? Math.min(100, Math.max(1, parsed.maxItems)) : 25
    state.randomSeedGuid = typeof parsed.randomSeedGuid === 'string' ? parsed.randomSeedGuid : ''
    state.randomSeedOffset = typeof parsed.randomSeedOffset === 'number'
      ? Math.max(0, Math.trunc(parsed.randomSeedOffset))
      : 0
    state.rootGroup = parsed.rootGroup && typeof parsed.rootGroup === 'object'
      ? normalizeGroupState(parsed.rootGroup as FilterGroupState)
      : createGroup('and')
  } catch {
    state.maxItems = 25
    state.randomSeedGuid = ''
    state.randomSeedOffset = 0
    state.rootGroup = createGroup('and')
  }
}

function normalizeGroupState(group: FilterGroupState): FilterGroupState {
  return {
    id: group.id || crypto.randomUUID(),
    combinator: group.combinator === 'or' ? 'or' : 'and',
    negated: group.negated === true,
    items: Array.isArray(group.items)
      ? group.items.map(item => item.kind === 'group'
        ? {
            kind: 'group',
            group: normalizeGroupState(item.group),
          }
        : {
            kind: 'rule',
            rule: item.rule,
          })
      : [],
  }
}
</script>

<template>
  <section class="page-grid builder-grid">
    <article class="panel panel-pad stack">
      <div class="cluster">
        <div class="chip">Datasource: {{ activeDatasource?.label || datasourceId }}</div>
        <div class="chip">Source: {{ state.source }}</div>
      </div>
      <p class="muted">
        The logic is explicit now: each group can be <code>Match all</code> (`AND`) or <code>Match any</code> (`OR`), and each group can also be negated with <code>NOT</code>.
        Each rule can be <code>Include</code> or <code>Exclude</code>, and you can add nested groups for parentheses-style logic.
      </p>
      <div class="cluster">
        <button class="button secondary" type="button" @click="resetBuilder">Clear builder</button>
      </div>
      <p v-if="loading" class="muted">Loading field metadata…</p>
      <p v-else-if="error" class="muted">{{ error }}</p>
      <FilterGroupEditor
        v-else
        :group="state.rootGroup"
        :fields="fields"
        :datasource-id="state.datasourceId"
        :source="state.source"
        :create-group="createGroup"
        :create-rule="createRule"
        :is-root="true"
        :depth="0"
      />
    </article>

    <aside class="side-panel stack">
      <JsonRequestPreview
        :request="activeRequest"
        :compare-status="payloadCompareStatus"
        :compare-message="payloadCompareMessage"
        @apply-payload="applyOutgoingPayload"
      />
      <QueryPreviewPanel
        :request="activeRequest"
        :max-items="state.maxItems"
        :random-seed-guid="state.randomSeedGuid"
        :random-seed-offset="state.randomSeedOffset"
        @update:max-items="updateMaxItems"
        @update:random-seed-guid="updateRandomSeedGuid"
        @update:random-seed-offset="updateRandomSeedOffset"
      />
    </aside>
  </section>
</template>
