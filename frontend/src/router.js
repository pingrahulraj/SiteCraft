import { createRouter, createWebHistory } from 'vue-router'

export const router = createRouter({
  history: createWebHistory(),
  routes: [
    { path: '/', redirect: '/login' },
    { path: '/login', component: () => import('./views/LoginView.vue') },
    { path: '/signup', component: () => import('./views/SignupView.vue') },
    { path: '/dashboard', component: () => import('./views/DashboardView.vue'), meta: { requiresAuth: true } },
    { path: '/site/:id', component: () => import('./views/SiteDashboardView.vue'), props: true, meta: { requiresAuth: true } },
    { path: '/site/:id/editor', component: () => import('./views/EditorView.vue'), props: true, meta: { requiresAuth: true } },
    { path: '/blogs', redirect: '/blogs/posts', meta: { requiresAuth: true } },
    { path: '/blogs/posts', component: () => import('./views/BlogsView.vue'), meta: { requiresAuth: true } },
    { path: '/blogs/posts/new', component: () => import('./views/BlogsView.vue'), meta: { requiresAuth: true } },
    { path: '/blogs/posts/:id/edit', component: () => import('./views/BlogsView.vue'), props: true, meta: { requiresAuth: true } },
    { path: '/blogs/categories', component: () => import('./views/BlogsView.vue'), meta: { requiresAuth: true } },
    { path: '/blogs/tags', component: () => import('./views/BlogsView.vue'), meta: { requiresAuth: true } },
    { path: '/blogs/authors', component: () => import('./views/BlogsView.vue'), meta: { requiresAuth: true } },
    { path: '/blogs/media', component: () => import('./views/BlogsView.vue'), meta: { requiresAuth: true } },
    { path: '/blogs/comments', component: () => import('./views/BlogsView.vue'), meta: { requiresAuth: true } },
    { path: '/blogs/newsletter', component: () => import('./views/BlogsView.vue'), meta: { requiresAuth: true } },
    { path: '/blogs/members', component: () => import('./views/BlogsView.vue'), meta: { requiresAuth: true } },
    { path: '/blogs/settings', component: () => import('./views/BlogsView.vue'), meta: { requiresAuth: true } }
  ]
})

router.beforeEach((to) => {
  const token = localStorage.getItem('sitecraft_token')
  if (to.meta.requiresAuth && !token) return '/login'
  if ((to.path === '/login' || to.path === '/signup') && token) return '/dashboard'
})
