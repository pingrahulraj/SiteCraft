<template>
  <main class="auth-shell">
    <section class="auth-layout">
      <div class="auth-hero">
        <div class="brand"><span class="brand-mark"></span> SiteCraft</div>
        <p class="eyebrow">Website builder platform</p>
        <h1 class="title">Code in HTML. Ship a clean preview URL. Keep the workflow focused.</h1>
        <p class="lead">A professional builder experience for designers and developers, with Carbon-inspired clarity and a lightweight architecture that can scale from SQLite to production.</p>
        <div class="feature">
          <div>
            <strong>Fast previewing</strong>
            <p class="lead">Generate a dedicated website URL for every project.</p>
          </div>
          <div>
            <strong>Maintainable stack</strong>
            <p class="lead">Vue 3 frontend with a Go API and clear separation of concerns.</p>
          </div>
        </div>
      </div>
      <div class="auth-panel">
        <div class="stack">
          <h2>Welcome back</h2>
          <p class="lead">Sign in to your workspace.</p>
          <p v-if="error" class="error">{{ error }}</p>
          <form class="form" @submit.prevent="submit">
            <input v-model="identifier" class="input" type="text" placeholder="Username or email" autocomplete="username" />
            <input v-model="password" class="input" type="password" placeholder="Password" autocomplete="current-password" />
            <button class="button" :disabled="loading">{{ loading ? 'Signing in...' : 'Log in' }}</button>
          </form>
          <router-link class="button secondary" to="/signup">Create account</router-link>
        </div>
      </div>
    </section>
  </main>
</template>

<script setup>
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { api } from '../services/api'

const identifier = ref('')
const password = ref('')
const loading = ref(false)
const error = ref('')
const router = useRouter()

const submit = async () => {
  loading.value = true
  error.value = ''
  try {
    const { data } = await api.post('/auth/login', {
      identifier: identifier.value,
      password: password.value
    })
    localStorage.setItem('sitecraft_token', data.token)
    await router.push('/dashboard')
  } catch (err) {
    error.value = err?.response?.data?.error === 'invalid_credentials'
      ? 'Incorrect email or password.'
      : 'Login failed. Please try again.'
  } finally {
    loading.value = false
  }
}
</script>
