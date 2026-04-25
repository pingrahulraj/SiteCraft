import { createRouter, createWebHistory } from 'vue-router'

export const router = createRouter({
  history: createWebHistory(),
  routes: [
    { path: '/', redirect: '/login' },
    { path: '/login', component: () => import('./views/LoginView.vue') },
    { path: '/signup', component: () => import('./views/SignupView.vue') },
    { path: '/dashboard', component: () => import('./views/DashboardView.vue') },
    { path: '/site/:id', component: () => import('./views/SiteDashboardView.vue'), props: true },
    { path: '/site/:id/editor', component: () => import('./views/EditorView.vue'), props: true },
    { path: '/blogs', redirect: '/blogs/posts' },
    { path: '/blogs/posts', component: () => import('./views/BlogsView.vue') },
    { path: '/blogs/posts/new', component: () => import('./views/BlogsView.vue') },
    { path: '/blogs/posts/:id/edit', component: () => import('./views/BlogsView.vue'), props: true },
    { path: '/blogs/categories', component: () => import('./views/BlogsView.vue') },
    { path: '/blogs/tags', component: () => import('./views/BlogsView.vue') },
    { path: '/blogs/authors', component: () => import('./views/BlogsView.vue') },
    { path: '/blogs/media', component: () => import('./views/BlogsView.vue') },
    { path: '/blogs/comments', component: () => import('./views/BlogsView.vue') },
    { path: '/blogs/newsletter', component: () => import('./views/BlogsView.vue') },
    { path: '/blogs/members', component: () => import('./views/BlogsView.vue') },
    { path: '/blogs/settings', component: () => import('./views/BlogsView.vue') }
  ]
})

router.beforeEach((to) => {
  const token = localStorage.getItem('sitecraft_token')
  if (to.path === '/dashboard' && !token) return '/login'
  if (to.path.startsWith('/site/') && !token) return '/login'
   if (to.path.startsWith('/blogs') && !token) return '/login'
  if ((to.path === '/login' || to.path === '/signup') && token) return '/dashboard'
})
