import axios from 'axios'

export const api = axios.create({
  baseURL: 'http://localhost:8080/api'
})

api.interceptors.request.use((config) => {
  const token = localStorage.getItem('sitecraft_token')
  if (token) config.headers.Authorization = `Bearer ${token}`
  return config
})

api.interceptors.response.use(
  (response) => response,
  (error) => {
    const status = error?.response?.status
    if (status === 401) {
      localStorage.removeItem('sitecraft_token')
      localStorage.removeItem('sitecraft_active_site_name')
      if (window.location.pathname !== '/login') {
        window.location.href = '/login'
      }
    } else if (status === 403) {
      window.dispatchEvent(new CustomEvent('sitecraft:forbidden', {
        detail: error?.response?.data?.error?.message || 'Access denied'
      }))
    }
    return Promise.reject(error)
  }
)
