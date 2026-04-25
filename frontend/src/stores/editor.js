import { defineStore } from 'pinia'
import { api } from '../services/api'

const nowIso = () => new Date().toISOString()
const defaultFileContent = (type, name) => {
  if (type === 'css') return `/* ${name} */\n`
  if (type === 'js') return `console.log('${name} loaded')\n`
  if (type === 'json') return '{\n  "name": "value"\n}\n'
  return '<section>New content</section>\n'
}

export const useEditorStore = defineStore('editor', {
  state: () => ({
    site: { id: null, name: 'Untitled site', status: 'Draft', publicUrl: '' },
    files: [],
    folders: [],
    openTabs: [],
    activePath: '/index.html',
    validationErrors: [],
    previewDevice: 'desktop',
    previewLoading: false,
    saveLoading: false,
    publishLoading: false,
    lastSavedAt: null,
    publishStatus: 'Not published',
    toast: null,
    contextMenu: null,
    publishResult: null,
    loading: false,
    error: ''
  }),
  getters: {
    activeFile(state) {
      return state.files.find((file) => file.path === state.activePath) ?? state.files[0] ?? null
    },
    dirtyCount(state) {
      return state.files.filter((file) => file.dirty).length
    },
    validationCount(state) {
      return state.validationErrors.length
    },
    previewUrl(state) {
      return state.site.publicUrl || ''
    },
    fileMap(state) {
      return state.files.reduce((acc, file) => {
        acc[file.path] = file
        return acc
      }, {})
    }
  },
  actions: {
    hydrate(siteId, siteName) {
      this.site.id = siteId
      if (siteName) this.site.name = siteName
    },
    async loadSite(siteId) {
      this.error = ''
      const { data } = await api.get(`/projects/${siteId}`)
      const site = data?.item || data
      this.site = {
        id: site.id,
        name: site.name || 'Untitled site',
        status: site.status || 'Draft',
        publicUrl: site.previewUrl || ''
      }
    },
    async loadWorkspace(siteId) {
      this.loading = true
      this.error = ''
      try {
        const { data } = await api.get(`/sites/${siteId}/workspace/files`)
        const items = data.items || []
        const files = []
        for (const item of items) {
          const contentResponse = await api.get(`/sites/${siteId}/workspace/files/content`, { params: { path: item.path } })
          files.push({
            id: item.path,
            name: item.name || item.path.split('/').pop(),
            path: item.path.startsWith('/') ? item.path : `/${item.path}`,
            type: item.fileType || 'txt',
            dirty: false,
            content: contentResponse.data?.item?.content || ''
          })
        }
        this.files = files.length ? files : [{
          id: '/index.html',
          name: 'index.html',
          path: '/index.html',
          type: 'html',
          dirty: false,
          content: '<!doctype html><html><body><main></main></body></html>'
        }]
        this.openTabs = this.files.slice(0, 3).map((file) => file.path)
        if (!this.fileMap[this.activePath]) this.activePath = this.files[0].path
      } catch (error) {
        this.error = error.response?.data?.error?.message || 'The editor workspace could not be loaded.'
        this.files = [{
          id: '/index.html',
          name: 'index.html',
          path: '/index.html',
          type: 'html',
          dirty: false,
          content: '<!doctype html><html><body><main></main></body></html>'
        }]
        this.openTabs = ['/index.html']
        this.activePath = '/index.html'
      } finally {
        this.loading = false
      }
    },
    setActive(path) {
      this.activePath = path
      if (!this.openTabs.includes(path)) this.openTabs.push(path)
    },
    updateFile(path, content) {
      const file = this.files.find((item) => item.path === path)
      if (!file) return
      file.content = content
      file.dirty = true
    },
    async saveActive(siteId) {
      const file = this.activeFile
      if (!file) return
      this.saveLoading = true
      try {
        const { data } = await api.put(`/sites/${siteId}/workspace/files/content`, {
          path: file.path.replace(/^\//, ''),
          content: file.content
        })
        if (data?.success) {
          file.dirty = false
          this.lastSavedAt = nowIso()
          this.site.status = 'Saved'
          this.toast = { type: 'success', message: `${file.name} saved` }
        }
      } finally {
        this.saveLoading = false
      }
    },
    validate() {
      const errors = []
      const paths = new Set()
      const validPath = /^[a-zA-Z0-9/_\-.]+$/
      for (const file of this.files) {
        const barePath = file.path.replace(/^\//, '')
        if (!validPath.test(barePath)) {
          errors.push({ path: file.path, file: file.name, message: 'Invalid file path' })
        }
        if (paths.has(file.path)) {
          errors.push({ path: file.path, file: file.name, message: 'Duplicate path detected' })
        }
        paths.add(file.path)
        if (file.path === '/index.html' && !file.content.trim()) {
          errors.push({ path: file.path, file: file.name, message: 'index.html cannot be empty' })
        }
      }
      if (!paths.has('/index.html')) {
        errors.push({ path: '/index.html', file: 'index.html', message: 'index.html is required' })
      }
      this.validationErrors = errors
      return errors
    },
    async publish(siteId) {
      const errors = this.validate()
      if (errors.length) return { ok: false, errors }
      this.publishLoading = true
      this.publishStatus = 'Publishing'
      try {
        if (this.activeFile?.dirty) {
          await this.saveActive(siteId)
        }
        const { data } = await api.post(`/sites/${siteId}/publish`)
        if (!data?.success) return { ok: false, error: data?.error || 'publish_failed' }
        this.site.publicUrl = data.public_url
        this.publishResult = { title: 'Your site is live', url: data.public_url }
        this.publishStatus = 'Live'
        this.site.status = 'Live'
        this.toast = { type: 'success', message: data.message || 'Website published successfully' }
        return { ok: true, url: data.public_url }
      } finally {
        this.publishLoading = false
      }
    },
    async addFile(siteId, kind = 'html') {
      const ext = kind === 'css' ? 'css' : kind === 'js' ? 'js' : kind === 'json' ? 'json' : 'html'
      const baseName = kind === 'html' ? `page-${Date.now()}.${ext}` : kind === 'css' ? `styles-${Date.now()}.${ext}` : `script-${Date.now()}.${ext}`
      const payload = { path: `content/${baseName}`, fileType: ext, content: defaultFileContent(ext, baseName) }
      await api.post(`/sites/${siteId}/workspace/files`, payload)
      await this.loadWorkspace(siteId)
      this.setActive(`/content/${baseName}`)
    },
    async duplicateFile(siteId, path) {
      await api.post(`/sites/${siteId}/workspace/files/duplicate`, { path: path.replace(/^\//, '') })
      await this.loadWorkspace(siteId)
    },
    async renameFile(siteId, path, nextName) {
      const current = path.replace(/^\//, '')
      const parts = current.split('/')
      parts[parts.length - 1] = nextName
      await api.put(`/sites/${siteId}/workspace/files/rename`, {
        path: current,
        newPath: parts.join('/')
      })
      await this.loadWorkspace(siteId)
      this.setActive(`/${parts.join('/')}`)
    },
    async deleteFile(siteId, path) {
      if (path === '/index.html') return false
      await api.delete(`/sites/${siteId}/workspace/files`, { data: { path: path.replace(/^\//, '') } })
      await this.loadWorkspace(siteId)
      this.activePath = '/index.html'
      return true
    },
    showToast(type, message) {
      this.toast = { type, message }
    },
    clearToast() {
      this.toast = null
    }
  }
})
