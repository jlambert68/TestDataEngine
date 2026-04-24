import { createRouter, createWebHistory } from 'vue-router'

import HomePage from './pages/HomePage.vue'
import BuilderPage from './pages/BuilderPage.vue'

export default createRouter({
  history: createWebHistory(),
  routes: [
    { path: '/', component: HomePage },
    { path: '/builder/:datasourceId', component: BuilderPage, props: true },
  ],
})
