import { defineStore } from 'pinia'
import { api } from '../services/api'

const blankPost = () => ({
  id: null,
  title: '',
  slug: '',
  excerpt: '',
  content: '',
  featuredImage: '',
  author: '',
  authorId: '',
  tags: '',
  seoTitle: '',
  seoDescription: '',
  visibility: 'public',
  status: 'draft',
  publishedAt: '',
  scheduledAt: ''
})

const blankSettings = () => ({
  blogTitle: '',
  description: '',
  defaultAuthorId: '',
  rssEnabled: true,
  commentsEnabled: true,
  canonicalUrl: '',
  postsPerPage: 10,
  socialSharingEnabled: true
})

const toArray = (value) => Array.isArray(value) ? value : []
const toItem = (value) => value?.item || value || {}

export const useBlogsStore = defineStore('blogs', {
  state: () => ({
    posts: [],
    categories: [],
    tags: [],
    authors: [],
    comments: [],
    newsletterSettings: null,
    blogSettings: blankSettings(),
    members: [],
    analytics: null,
    loading: false,
    saving: false,
    error: '',
    feedback: '',
    siteId: '',
    selected: blankPost()
  }),
  actions: {
    clearFeedback() {
      this.feedback = ''
    },
    setSite(siteId) {
      this.siteId = siteId
    },
    select(post) {
      this.selected = post ? { ...blankPost(), ...post } : blankPost()
    },
    async load(siteId) {
      this.loading = true
      this.error = ''
      this.siteId = siteId
      try {
        const [posts, categories, tags, authors, newsletter, members, analytics, settings, comments] = await Promise.all([
          api.get(`/sites/${siteId}/posts`),
          api.get(`/sites/${siteId}/blog-categories`),
          api.get(`/sites/${siteId}/blog-tags`),
          api.get(`/sites/${siteId}/blog-authors`),
          api.get(`/sites/${siteId}/newsletter-settings`),
          api.get(`/sites/${siteId}/blog-members`),
          api.get(`/sites/${siteId}/blog-analytics`),
          api.get(`/sites/${siteId}/blog-settings`),
          api.get(`/sites/${siteId}/blog-comments`)
        ])
        this.posts = toArray(posts.data?.items)
        this.categories = toArray(categories.data?.items)
        this.tags = toArray(tags.data?.items)
        this.authors = toArray(authors.data?.items)
        this.newsletterSettings = toItem(newsletter.data)
        this.members = toArray(members.data?.items)
        this.analytics = toItem(analytics.data)
        this.blogSettings = { ...blankSettings(), ...toItem(settings.data) }
        this.comments = toArray(comments.data?.items)
      } catch (error) {
        this.error = error.response?.data?.error?.message || 'Failed to load blog workspace'
      } finally {
        this.loading = false
      }
    },
    async loadPost(siteId, postId) {
      this.saving = true
      this.error = ''
      try {
        const { data } = await api.get(`/sites/${siteId}/posts/${postId}`)
        const item = toItem(data)
        this.select(item)
        return item
      } catch (error) {
        this.error = error.response?.data?.error?.message || 'Failed to load post'
        return null
      } finally {
        this.saving = false
      }
    },
    async save(siteId, payload) {
      this.saving = true
      this.error = ''
      this.feedback = ''
      try {
        let response
        if (payload.id) response = await api.put(`/sites/${siteId}/posts/${payload.id}`, payload)
        else response = await api.post(`/sites/${siteId}/posts`, payload)
        await this.load(siteId)
        const item = payload.id ? this.posts.find((post) => post.id === payload.id) : this.posts[0]
        this.select(item || payload)
        this.feedback = payload.id ? 'Post updated.' : 'Draft saved.'
        return response?.data
      } catch (error) {
        this.error = error.response?.data?.error?.message || 'Failed to save post'
        return null
      } finally {
        this.saving = false
      }
    },
    async remove(siteId, id) {
      await api.delete(`/sites/${siteId}/posts/${id}`)
      await this.load(siteId)
    },
    async trash(siteId, id) {
      await api.post(`/sites/${siteId}/posts/${id}/trash`)
      await this.load(siteId)
    },
    async duplicate(siteId, id) {
      await api.post(`/sites/${siteId}/posts/${id}/duplicate`)
      await this.load(siteId)
    },
    async setPublished(siteId, id, published) {
      const action = published ? 'publish' : 'unpublish'
      await api.post(`/sites/${siteId}/posts/${id}/${action}`)
      await this.load(siteId)
      this.feedback = published ? 'Post published.' : 'Post moved to draft.'
    },
    async saveCategory(siteId, payload) {
      if (payload.id) await api.put(`/sites/${siteId}/blog-categories/${payload.id}`, payload)
      else await api.post(`/sites/${siteId}/blog-categories`, payload)
      await this.load(siteId)
    },
    async removeCategory(siteId, id) {
      await api.delete(`/sites/${siteId}/blog-categories/${id}`)
      await this.load(siteId)
    },
    async saveTag(siteId, payload) {
      if (payload.id) await api.put(`/sites/${siteId}/blog-tags/${payload.id}`, payload)
      else await api.post(`/sites/${siteId}/blog-tags`, payload)
      await this.load(siteId)
    },
    async removeTag(siteId, id) {
      await api.delete(`/sites/${siteId}/blog-tags/${id}`)
      await this.load(siteId)
    },
    async saveAuthor(siteId, payload) {
      if (payload.id) await api.put(`/sites/${siteId}/blog-authors/${payload.id}`, payload)
      else await api.post(`/sites/${siteId}/blog-authors`, payload)
      await this.load(siteId)
    },
    async removeAuthor(siteId, id) {
      await api.delete(`/sites/${siteId}/blog-authors/${id}`)
      await this.load(siteId)
    },
    async saveNewsletterSettings(siteId, payload) {
      await api.put(`/sites/${siteId}/newsletter-settings`, payload)
      await this.load(siteId)
      this.feedback = 'Newsletter settings saved.'
    },
    async saveBlogSettings(siteId, payload) {
      await api.put(`/sites/${siteId}/blog-settings`, payload)
      await this.load(siteId)
      this.feedback = 'Blog settings updated.'
    },
    async saveMember(siteId, payload) {
      if (payload.id) await api.put(`/sites/${siteId}/blog-members/${payload.id}`, payload)
      else await api.post(`/sites/${siteId}/blog-members`, payload)
      await this.load(siteId)
    },
    async removeMember(siteId, id) {
      await api.delete(`/sites/${siteId}/blog-members/${id}`)
      await this.load(siteId)
    },
    async moderateComment(siteId, id, action) {
      await api.post(`/sites/${siteId}/blog-comments/${id}/${action}`)
      await this.load(siteId)
    }
  }
})

export { blankPost, blankSettings }
