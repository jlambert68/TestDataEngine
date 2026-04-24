<script setup lang="ts">
import { computed } from 'vue'
import { useRoute } from 'vue-router'

const route = useRoute()

const datasourcesLink = computed(() => {
  if (route.path.startsWith('/builder/')) {
    return {
      path: '/',
      query: {
        returnTo: route.fullPath,
      },
    }
  }
  return { path: '/' }
})
</script>

<template>
  <div class="app-shell">
    <header class="app-header">
      <div>
        <p class="eyebrow">TestDataEngine</p>
        <h1>Filter Builder</h1>
      </div>
      <router-link class="ghost-link" :to="datasourcesLink">Datasources</router-link>
    </header>
    <main class="app-main">
      <router-view />
    </main>
  </div>
</template>
