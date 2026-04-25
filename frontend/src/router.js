import { createRouter, createWebHistory } from 'vue-router'

export const router = createRouter({
  history: createWebHistory(),
  routes: [
    { path: '/', redirect: '/login' },
    { path: '/login', component: () => import('./views/LoginView.vue') },
    { path: '/signup', component: () => import('./views/SignupView.vue') },
    { path: '/dashboard', component: () => import('./views/DashboardView.vue') },
    { path: '/site/:id', component: () => import('./views/SiteDashboardView.vue'), props: true },
    { path: '/site/:id/editor', component: () => import('./views/EditorView.vue'), props: true }
  ]
})

router.beforeEach((to) => {
  const token = localStorage.getItem('sitecraft_token')
  if (to.path === '/dashboard' && !token) return '/login'
  if (to.path.startsWith('/site/') && !token) return '/login'
  if ((to.path === '/login' || to.path === '/signup') && token) return '/dashboard'
})
