import { createApp } from 'vue'
import { createPinia } from 'pinia'
import { router } from './router'
import App from './App.vue'
import './styles.css'

const app = createApp(App)

app.config.errorHandler = (error) => {
  console.error(error)
  window.dispatchEvent(new CustomEvent('sitecraft:fatal', {
    detail: error?.message || 'Unexpected application error'
  }))
}

app.use(createPinia()).use(router).mount('#app')
