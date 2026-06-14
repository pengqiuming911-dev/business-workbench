import { createRouter, createWebHistory, type RouteRecordRaw } from 'vue-router'
import Dashboard from '../views/Dashboard.vue'

const routes: RouteRecordRaw[] = [
  { path: '/', component: Dashboard },
  { path: '/data-preparation', component: () => import('../views/DataPreparation.vue') },
  { path: '/product-report', component: () => import('../views/ProductReport.vue') },
  { path: '/product-completion', component: () => import('../views/ProductCompletion.vue') },
  { path: '/holding-analysis', component: () => import('../views/HoldingAnalysis.vue') },
  { path: '/push-settings', component: () => import('../views/PushSettings.vue') },
  { path: '/activity-log', component: () => import('../views/ActivityLog.vue') },
  { path: '/agent', component: () => import('../views/AgentChat.vue') },
]

export default createRouter({
  history: createWebHistory(),
  routes,
})
