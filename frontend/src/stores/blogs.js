import { defineStore } from 'pinia'
import { api } from '../services/api'

const blankPost = () => ({
  id: null,
  title: '',
  slug: '',
  excerpt: '',
  content: '',
  featuredImage: '',
  author: 'Ava Chen',
  tags: '',
  seoTitle: '',
  seoDescription: '',
  visibility: 'public',
  status: 'draft'
})

export const useBlogsStore = defineStore('blogs', {
  state: () => ({
    posts: [],
    categories: [],
    tags: [],
    authors: [],
    newsletterSettings: null,
    members: [],
    analytics: null,
    loading: false,
    saving: false,
    error: '',
    selected: blankPost()
  }),
  actions: {
    async load(siteId) {
      this.loading = true
      this.error = ''
      try {
        const [posts, categories, tags, authors, newsletter, members, analytics] = await Promise.all([
          api.get(`/sites/${siteId}/posts`),
          api.get(`/sites/${siteId}/blog-categories`),
          api.get(`/sites/${siteId}/blog-tags`),
          api.get(`/sites/${siteId}/blog-authors`),
          api.get(`/sites/${siteId}/newsletter-settings`),
          api.get(`/sites/${siteId}/blog-members`),
          api.get(`/sites/${siteId}/blog-analytics`)
        ])
        this.posts = posts.data.items || []
        this.categories = categories.data.items || []
        this.tags = tags.data.items || []
        this.authors = authors.data.items || []
        this.newsletterSettings = newsletter.data.item || null
        this.members = members.data.items || []
        this.analytics = analytics.data.item || null
      } catch (error) {
        this.error = error.response?.data?.error?.message || 'Failed to load blog workspace'
      } finally {
        this.loading = false
      }
    },
    select(post) {
      this.selected = post ? { ...post } : blankPost()
    },
    async save(siteId, payload) {
      this.saving = true
      this.error = ''
      try {
        if (payload.id) {
          await api.put(`/sites/${siteId}/posts/${payload.id}`, payload)
        } else {
          await api.post(`/sites/${siteId}/posts`, payload)
        }
        await this.load(siteId)
        this.select()
      } catch (error) {
        this.error = error.response?.data?.error?.message || 'Failed to save post'
      } finally {
        this.saving = false
      }
    },
    async remove(siteId, id) {
      await api.delete(`/sites/${siteId}/posts/${id}`)
      await this.load(siteId)
    },
    async setPublished(siteId, id, published) {
      const action = published ? 'publish' : 'unpublish'
      await api.post(`/sites/${siteId}/posts/${id}/${action}`)
      await this.load(siteId)
    },
    async saveCategory(siteId, payload) {
      if (payload.id) await api.put(`/sites/${siteId}/blog-categories/${payload.id}`, payload)
      else await api.post(`/sites/${siteId}/blog-categories`, payload)
      await this.load(siteId)
    },
    async saveTag(siteId, payload) {
      if (payload.id) await api.put(`/sites/${siteId}/blog-tags/${payload.id}`, payload)
      else await api.post(`/sites/${siteId}/blog-tags`, payload)
      await this.load(siteId)
    },
    async saveAuthor(siteId, payload) {
      if (payload.id) await api.put(`/sites/${siteId}/blog-authors/${payload.id}`, payload)
      else await api.post(`/sites/${siteId}/blog-authors`, payload)
      await this.load(siteId)
    },
    async saveNewsletterSettings(siteId, payload) {
      await api.put(`/sites/${siteId}/newsletter-settings`, payload)
      await this.load(siteId)
    },
    async saveMember(siteId, payload) {
      if (payload.id) await api.put(`/sites/${siteId}/blog-members/${payload.id}`, payload)
      else await api.post(`/sites/${siteId}/blog-members`, payload)
      await this.load(siteId)
    }
  }
})
