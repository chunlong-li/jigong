import { createRouter, createWebHistory } from 'vue-router'
import { api } from '../api'

const router = createRouter({
  history: createWebHistory(),
  routes: [
    { path: '/login', name: 'login', component: () => import('../views/LoginView.vue') },
    { path: '/:pathMatch(.*)*', redirect: '/' },
    {
      path: '/',
      component: () => import('../layouts/MainLayout.vue'),
      children: [
        { path: '', name: 'dashboard', component: () => import('../views/DashboardView.vue') },
        { path: 'projects', name: 'projects', component: () => import('../views/ProjectsView.vue') },
        { path: 'workers', name: 'workers', component: () => import('../views/WorkersView.vue') },
        { path: 'worklogs', name: 'worklogs', component: () => import('../views/WorkLogsView.vue') },
        { path: 'prices', name: 'prices', component: () => import('../views/PricesView.vue') },
        { path: 'quotations', name: 'quotations', component: () => import('../views/QuotationsView.vue') },
        { path: 'settlements', name: 'settlements', component: () => import('../views/SettlementsView.vue') },
        { path: 'settings', name: 'settings', component: () => import('../views/SettingsView.vue') },
      ],
    },
  ],
})

let checked = false

router.beforeEach(async (to) => {
  if (to.name === 'login') return true
  if (checked) return true
  try {
    await api.me()
    checked = true
    return true
  } catch {
    return { name: 'login', query: { redirect: to.fullPath } }
  }
})

export default router
