import { createRouter, createWebHistory } from 'vue-router'
import Dashboard from '../views/Dashboard.vue'

const routes = [
  { path: '/', component: Dashboard },
  { path: '/data-preparation', component: () => import('../views/DataPreparation.vue') },
  { path: '/product-report', component: () => import('../views/ProductReport.vue') },
  { path: '/product-completion', component: () => import('../views/ProductCompletion.vue') },
  { path: '/ongoing-product', component: () => import('../views/OngoingProduct.vue') },
  { path: '/push-settings', component: () => import('../views/PushSettings.vue') },
  { path: '/activity-log', component: () => import('../views/ActivityLog.vue') },
  { path: '/agent', component: () => import('../views/AgentChat.vue') },
]

export default createRouter({
  history: createWebHistory(),
  routes
})