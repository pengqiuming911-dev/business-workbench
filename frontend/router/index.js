import { createRouter, createWebHistory } from 'vue-router'
import Home from '../views/Home.vue'

const routes = [
  { path: '/', component: Home },
  { path: '/data-preparation', component: () => import('../views/DataPreparation.vue') },
  { path: '/customer-churn', component: () => import('../views/CustomerChurn.vue') },
  { path: '/product-report', component: () => import('../views/ProductReport.vue') },
  { path: '/product-completion', component: () => import('../views/ProductCompletion.vue') },
  { path: '/ongoing-product', component: () => import('../views/OngoingProduct.vue') },
  { path: '/channel-analysis', component: () => import('../views/ChannelAnalysis.vue') },
  { path: '/nominal-buyer', component: () => import('../views/NominalBuyer.vue') },
  { path: '/user-profile', component: () => import('../views/UserProfile.vue') },
]

export default createRouter({
  history: createWebHistory(),
  routes
})
