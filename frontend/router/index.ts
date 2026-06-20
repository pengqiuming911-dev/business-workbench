import { createRouter, createWebHistory, type RouteRecordRaw } from 'vue-router'
import Dashboard from '../views/Dashboard.vue'

const routes: RouteRecordRaw[] = [
  { path: '/', component: Dashboard },
  { path: '/data-preparation', component: () => import('../views/DataPreparation.vue') },
  { path: '/product-report', component: () => import('../views/ProductReport.vue') },
  { path: '/product-completion', component: () => import('../views/ProductCompletion.vue') },
  { path: '/holding-analysis', component: () => import('../views/HoldingAnalysis.vue') },
  { path: '/product-analysis', redirect: '/holding-analysis?tab=product' },
  { path: '/customer-holding', redirect: '/holding-analysis?tab=customer' },
  { path: '/rebate-analysis', component: () => import('../views/RebateAnalysis.vue') },
  { path: '/rebate-pending', redirect: '/rebate-analysis?tab=pending' },
  { path: '/rebate-completed', redirect: '/rebate-analysis?tab=completed' },
  { path: '/ongoing-product', redirect: '/holding-analysis' },
  { path: '/push-settings', component: () => import('../views/PushSettings.vue') },
  { path: '/channel-analysis', redirect: '/' },
  { path: '/activity-log', component: () => import('../views/ActivityLog.vue') },
  { path: '/user-profile', component: () => import('../views/UserProfile.vue') },
  { path: '/agent', component: () => import('../views/AgentChat.vue') },
]

export default createRouter({
  history: createWebHistory(),
  routes,
})
