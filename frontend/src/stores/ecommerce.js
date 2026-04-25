import { defineStore } from 'pinia'
import { api } from '../services/api'

const blankProduct = () => ({
  id: null,
  name: '',
  slug: '',
  description: '',
  price: 0,
  sku: '',
  imageUrl: '',
  category: '',
  stockStatus: 'in_stock',
  seoTitle: '',
  seoDescription: '',
  status: 'draft'
})

export const useEcommerceStore = defineStore('ecommerce', {
  state: () => ({
    products: [],
    orders: [],
    customers: [],
    coupons: [],
    shippingZones: [],
    storeSettings: null,
    taxSettings: null,
    loading: false,
    saving: false,
    error: '',
    selected: blankProduct()
  }),
  actions: {
    async load(siteId) {
      this.loading = true
      this.error = ''
      try {
        const [products, orders, customers, coupons, shippingZones, storeSettings, taxSettings] = await Promise.all([
          api.get(`/sites/${siteId}/products`),
          api.get(`/sites/${siteId}/orders`),
          api.get(`/sites/${siteId}/customers`),
          api.get(`/sites/${siteId}/coupons`),
          api.get(`/sites/${siteId}/shipping-zones`),
          api.get(`/sites/${siteId}/store-settings`),
          api.get(`/sites/${siteId}/tax-settings`)
        ])
        this.products = products.data.items || []
        this.orders = orders.data.items || []
        this.customers = customers.data.items || []
        this.coupons = coupons.data.items || []
        this.shippingZones = shippingZones.data.items || []
        this.storeSettings = storeSettings.data.item || null
        this.taxSettings = taxSettings.data.item || null
      } catch (error) {
        this.error = error.response?.data?.error?.message || 'Failed to load ecommerce workspace'
      } finally {
        this.loading = false
      }
    },
    select(product) {
      this.selected = product ? { ...product } : blankProduct()
    },
    async save(siteId, payload) {
      this.saving = true
      this.error = ''
      try {
        if (payload.id) {
          await api.put(`/sites/${siteId}/products/${payload.id}`, payload)
        } else {
          await api.post(`/sites/${siteId}/products`, payload)
        }
        await this.load(siteId)
        this.select()
      } catch (error) {
        this.error = error.response?.data?.error?.message || 'Failed to save product'
      } finally {
        this.saving = false
      }
    },
    async remove(siteId, id) {
      await api.delete(`/sites/${siteId}/products/${id}`)
      await this.load(siteId)
    },
    async duplicate(siteId, id) {
      await api.post(`/sites/${siteId}/products/${id}/duplicate`)
      await this.load(siteId)
    },
    async saveCoupon(siteId, payload) {
      if (payload.id) await api.put(`/sites/${siteId}/coupons/${payload.id}`, payload)
      else await api.post(`/sites/${siteId}/coupons`, payload)
      await this.load(siteId)
    },
    async saveShippingZone(siteId, payload) {
      if (payload.id) await api.put(`/sites/${siteId}/shipping-zones/${payload.id}`, payload)
      else await api.post(`/sites/${siteId}/shipping-zones`, payload)
      await this.load(siteId)
    },
    async saveStoreSettings(siteId, payload) {
      await api.put(`/sites/${siteId}/store-settings`, payload)
      await this.load(siteId)
    },
    async saveTaxSettings(siteId, payload) {
      await api.put(`/sites/${siteId}/tax-settings`, payload)
      await this.load(siteId)
    }
  }
})
