<template>
  <div>
    <div v-if="forbiddenMessage" class="app-alert">
      <span>{{ forbiddenMessage }}</span>
      <button type="button" class="app-alert-close" @click="forbiddenMessage = ''">Dismiss</button>
    </div>
    <div v-if="fatalError" class="app-fatal">
      <div class="surface-card">
        <p class="eyebrow">Application error</p>
        <h2>Something went wrong</h2>
        <p class="lead">{{ fatalError }}</p>
      </div>
    </div>
    <router-view v-else />
  </div>
</template>

<script setup>
import { onBeforeUnmount, onMounted, ref } from 'vue'

const forbiddenMessage = ref('')
const fatalError = ref('')

const onForbidden = (event) => {
  forbiddenMessage.value = event.detail || 'Access denied'
}

const onFatal = (event) => {
  fatalError.value = event.detail || 'Unexpected application error'
}

onMounted(() => {
  window.addEventListener('sitecraft:forbidden', onForbidden)
  window.addEventListener('sitecraft:fatal', onFatal)
})

onBeforeUnmount(() => {
  window.removeEventListener('sitecraft:forbidden', onForbidden)
  window.removeEventListener('sitecraft:fatal', onFatal)
})
</script>
