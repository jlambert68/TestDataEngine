<script setup lang="ts">
defineOptions({
  name: 'FilterGroupEditor',
})

import FieldFilterCard from './FieldFilterCard.vue'
import type { FieldDescriptor, FilterGroupState, FilterRuleState, SourceType } from '../../types/api'

const props = defineProps<{
  group: FilterGroupState
  fields: FieldDescriptor[]
  datasourceId: string
  source: SourceType
  createRule: (field?: string) => FilterRuleState
  createGroup: (combinator: 'and' | 'or') => FilterGroupState
  isRoot?: boolean
  depth?: number
}>()

const emit = defineEmits<{
  remove: []
}>()

function addRule() {
  props.group.items.push({
    kind: 'rule',
    rule: props.createRule(),
  })
}

function addGroup() {
  props.group.items.push({
    kind: 'group',
    group: props.createGroup('and'),
  })
}

function removeAt(index: number) {
  props.group.items.splice(index, 1)
}
</script>

<template>
  <section class="group-shell stack" :style="{ marginLeft: isRoot ? '0' : `${(depth ?? 0) * 18}px` }">
    <header class="group-header">
      <div class="stack" style="gap: 10px;">
        <div class="cluster">
          <span class="chip">{{ isRoot ? 'Root group' : `Nested group level ${(depth ?? 0) + 1}` }}</span>
          <span class="chip">{{ group.items.length }} items</span>
        </div>
        <div class="cluster">
          <label class="chip">
            <input :checked="group.combinator === 'and'" type="radio" :name="`group-mode-${group.id}`" @change="group.combinator = 'and'" />
            Match all (AND)
          </label>
          <label class="chip">
            <input :checked="group.combinator === 'or'" type="radio" :name="`group-mode-${group.id}`" @change="group.combinator = 'or'" />
            Match any (OR)
          </label>
          <label class="chip">
            <input :checked="group.negated" type="checkbox" @change="group.negated = !group.negated" />
            Negate (NOT)
          </label>
        </div>
      </div>

      <button v-if="!isRoot" class="button secondary" type="button" @click="emit('remove')">Remove group</button>
    </header>

    <p class="muted">
      {{
        group.negated
          ? `NOT ${group.combinator === 'and' ? '(all items in this group must match)' : '(any item in this group may match)'}`
          : group.combinator === 'and'
            ? 'All items in this group must match.'
            : 'Any item in this group may match.'
      }}
    </p>

    <div v-if="group.items.length" class="stack">
      <div v-for="(item, index) in group.items" :key="item.kind === 'rule' ? item.rule.id : item.group.id">
        <FieldFilterCard
          v-if="item.kind === 'rule'"
          :datasource-id="datasourceId"
          :source="source"
          :fields="fields"
          :rule="item.rule"
          :removable="true"
          @remove="removeAt(index)"
        />

        <FilterGroupEditor
          v-else
          :group="item.group"
          :fields="fields"
          :datasource-id="datasourceId"
          :source="source"
          :create-group="createGroup"
          :create-rule="createRule"
          :depth="(depth ?? 0) + 1"
          @remove="removeAt(index)"
        />
      </div>
    </div>
    <p v-else class="muted">
      This group is empty. Add a rule for a field condition, or add a nested group for parentheses-style logic.
    </p>

    <footer class="cluster">
      <button class="button" type="button" @click="addRule">Add rule</button>
      <button class="button secondary" type="button" @click="addGroup">Add subgroup</button>
    </footer>
  </section>
</template>
