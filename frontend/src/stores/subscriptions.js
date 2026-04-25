import { defineStore } from 'pinia'
import { api } from '../services/api'

const blankPlan = () => ({
  id: null,
  name: '',
  description: '',
  price: 0,
  interval: 'monthly',
  trial: 14,
  status: 'draft'
})

export const useSubscriptionsStore = defineStore('subscriptions', {
  state: () => ({
    plans: [],
    subscribers: [],
    subscriptions: [],
    invoices: [],
    failedPayments: [],
    settings: null,
    loading: false,
    saving: false,
    error: '',
    selected: blankPlan()
  }),
  actions: {
    async load(siteId) {
      this.loading = true
      this.error = ''
      try {
        const [plans, subscribers, subscriptions, invoices, failedPayments, settings] = await Promise.all([
          api.get(`/sites/${siteId}/subscription-plans`),
          api.get(`/sites/${siteId}/subscription-subscribers`),
          api.get(`/sites/${siteId}/subscriptions`),
          api.get(`/sites/${siteId}/subscription-invoices`),
          api.get(`/sites/${siteId}/failed-payments`),
          api.get(`/sites/${siteId}/subscription-settings`)
        ])
        this.plans = plans.data.items || []
        this.subscribers = subscribers.data.items || []
        this.subscriptions = subscriptions.data.items || []
        this.invoices = invoices.data.items || []
        this.failedPayments = failedPayments.data.items || []
        this.settings = settings.data.item || null
      } catch (error) {
        this.error = error.response?.data?.error?.message || 'Failed to load subscriptions workspace'
      } finally {
        this.loading = false
      }
    },
    async savePlan(siteId, payload) {
      this.saving = true
      try {
        if (payload.id) await api.put(`/sites/${siteId}/subscription-plans/${payload.id}`, payload)
        else await api.post(`/sites/${siteId}/subscription-plans`, payload)
        await this.load(siteId)
      } finally {
        this.saving = false
      }
    },
    async removePlan(siteId, id) {
      await api.delete(`/sites/${siteId}/subscription-plans/${id}`)
      await this.load(siteId)
    },
    async saveSettings(siteId, payload) {
      await api.put(`/sites/${siteId}/subscription-settings`, payload)
      await this.load(siteId)
    },
    async retryFailedPayment(siteId, id) {
      await api.post(`/sites/${siteId}/failed-payments/${id}/retry`)
      await this.load(siteId)
    }
  }
})
