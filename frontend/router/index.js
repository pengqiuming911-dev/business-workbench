import { createRouter, createWebHistory } from 'vue-router'
import Dashboard from '../views/Dashboard.vue'

const routes = [
  { path: '/', component: Dashboard },
  { path: '/data-preparation', component: () => import('../views/DataPreparation.vue') },
  { path: '/customer-churn', component: () => import('../views/CustomerChurn.vue') },
  { path: '/product-report', component: () => import('../views/ProductReport.vue') },
  { path: '/product-completion', component: () => import('../views/ProductCompletion.vue') },
  { path: '/ongoing-product', component: () => import('../views/OngoingProduct.vue') },
  { path: '/channel-analysis', component: () => import('../views/ChannelAnalysis.vue') },
  { path: '/nominal-buyer', component: () => import('../views/NominalBuyer.vue') },
  { path: '/user-profile', component: () => import('../views/UserProfile.vue') },
  { path: '/activity-log', component: () => import('../views/ActivityLog.vue') },
]

export default createRouter({
  history: createWebHistory(),
  routes
})