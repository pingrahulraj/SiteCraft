<template>
  <main class="shell">
    <section class="card hero">
      <div class="panel">
        <div class="brand"><span class="brand-mark"></span> SiteCraft</div>
        <p class="eyebrow">Get started</p>
        <h1 class="title">Create your workspace in a UI that feels polished from day one.</h1>
        <p class="lead">Structured, calm, and modern. The layout is designed to support the product, not distract from it.</p>
      </div>
      <div class="panel auth">
        <div class="stack">
          <h2>Create account</h2>
          <form class="form" @submit.prevent="submit">
            <input v-model="tenantSlug" class="input" type="text" placeholder="Tenant slug" />
            <input v-model="username" class="input" type="text" placeholder="Username" autocomplete="username" />
            <input v-model="name" class="input" type="text" placeholder="Name" />
            <input v-model="email" class="input" type="email" placeholder="Email" />
            <input v-model="password" class="input" type="password" placeholder="Password" />
            <button class="button">Sign up</button>
          </form>
          <router-link class="button secondary" to="/login">Back to login</router-link>
        </div>
      </div>
    </section>
  </main>
</template>

<script setup>
import { ref } from 'vue'
import { api } from '../services/api'

const tenantSlug = ref('acme-studio')
const username = ref('')
const name = ref('')
const email = ref('')
const password = ref('')
const submit = async () => {
  await api.post('/auth/register', {
    tenantSlug: tenantSlug.value,
    username: username.value,
    name: name.value,
    email: email.value,
    password: password.value
  })
}
</script>
