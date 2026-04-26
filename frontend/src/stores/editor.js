import { defineStore } from 'pinia'
import { api } from '../services/api'

const nowIso = () => new Date().toISOString()
const SAFE_NAME = /^[a-zA-Z0-9][a-zA-Z0-9._-]*$/
const PROTECTED_PATHS = new Set(['/index.html'])

const defaultContentForType = (type, name) => {
  if (type === 'css') return `/* ${name} */\n:root {\n  color-scheme: light;\n}\n`
  if (type === 'js') return `document.addEventListener('DOMContentLoaded', () => {\n  console.log('${name} loaded')\n})\n`
  return '<section>\n  <h2>New page</h2>\n  <p>Start editing this page.</p>\n</section>\n'
}

const normalizeSegments = (value) => value.split('/').filter(Boolean)
const buildPath = (...parts) => '/' + parts.flatMap((part) => normalizeSegments(String(part || ''))).join('/')
const dirname = (path) => {
  const segments = normalizeSegments(path)
  segments.pop()
  return segments.length ? '/' + segments.join('/') : '/'
}
const basename = (path) => normalizeSegments(path).slice(-1)[0] || ''
const extensionForType = (type) => {
  if (type === 'css') return 'css'
  if (type === 'js') return 'js'
  return 'html'
}
const titleFromPath = (path) => {
  if (path === '/index.html') return 'Home'
  return basename(path).replace(/\.[^.]+$/, '').replace(/[-_]/g, ' ')
}

export const useEditorStore = defineStore('editor', {
  state: () => ({
    site: { id: '', name: 'Untitled site', status: 'Draft', publicUrl: '' },
    files: [],
    folders: [],
    openTabs: [],
    activePath: '/index.html',
    expandedFolders: ['/content', '/assets'],
    validationErrors: [],
    validationStatus: 'Not validated',
    previewDevice: 'desktop',
    previewLoading: false,
    previewNonce: 0,
    previewHtml: '',
    saveLoading: false,
    publishLoading: false,
    loading: false,
    error: '',
    lastSavedAt: '',
    lastPublishedAt: '',
    publishStatus: 'Draft only',
    publishResult: null,
    toast: null,
    cursor: { line: 1, column: 1 },
    dirtyPaths: {},
    contextTargetPath: ''
  }),
  getters: {
    fileMap(state) {
      return state.files.reduce((acc, file) => {
        acc[file.path] = file
        return acc
      }, {})
    },
    activeFile(state) {
      return state.files.find((file) => file.path === state.activePath) || state.files[0] || null
    },
    allFolders(state) {
      const known = new Set(['/'])
      state.folders.forEach((folder) => known.add(folder.path))
      state.files.forEach((file) => {
        let current = dirname(file.path)
        while (current && current !== '/' && !known.has(current)) {
          known.add(current)
          current = dirname(current)
        }
      })
      return Array.from(known)
        .filter((path) => path !== '/')
        .sort((a, b) => a.localeCompare(b))
        .map((path) => ({
          path,
          name: basename(path),
          kind: 'folder',
          protected: false,
          isPageGroup: path === '/content'
        }))
    },
    tree(state) {
      const folderMap = new Map()
      const addFolder = (folderPath) => {
        if (folderMap.has(folderPath)) return folderMap.get(folderPath)
        const node = {
          id: folderPath,
          path: folderPath,
          name: basename(folderPath),
          kind: 'folder',
          children: [],
          protected: false
        }
        folderMap.set(folderPath, node)
        const parent = dirname(folderPath)
        if (parent && parent !== '/' && folderPath !== '/') {
          addFolder(parent).children.push(node)
        }
        return node
      }

      this.allFolders.forEach((folder) => addFolder(folder.path))

      state.files.forEach((file) => {
        const node = {
          ...file,
          id: file.path,
          kind: 'file',
          protected: PROTECTED_PATHS.has(file.path),
          dirty: !!state.dirtyPaths[file.path]
        }
        const parent = dirname(file.path)
        if (parent !== '/' && folderMap.has(parent)) folderMap.get(parent).children.push(node)
      })

      const roots = []
      folderMap.forEach((node) => {
        if (dirname(node.path) === '/') roots.push(node)
      })

      const rootFiles = state.files
        .filter((file) => dirname(file.path) === '/')
        .map((file) => ({
          ...file,
          id: file.path,
          kind: 'file',
          protected: PROTECTED_PATHS.has(file.path),
          dirty: !!state.dirtyPaths[file.path]
        }))

      const sortNodes = (nodes) => nodes.sort((a, b) => {
        if (a.kind !== b.kind) return a.kind === 'folder' ? -1 : 1
        return a.name.localeCompare(b.name)
      }).map((node) => {
        if (node.kind === 'folder') node.children = sortNodes(node.children)
        return node
      })

      return sortNodes([...roots, ...rootFiles])
    },
    pageFiles(state) {
      return state.files.filter((file) => file.type === 'html')
    },
    dirtyCount(state) {
      return Object.values(state.dirtyPaths).filter(Boolean).length
    },
    previewUrl(state) {
      return state.site.publicUrl || ''
    },
    previewRequestUrl(state) {
      if (!state.site.id) return ''
      const path = encodeURIComponent(String(state.activePath || '/index.html').replace(/^\//, '') || 'index.html')
      return `${api.defaults.baseURL}/sites/${state.site.id}/preview?path=${path}&t=${state.previewNonce}`
    }
  },
  actions: {
    hydrate(siteId, siteName) {
      this.site.id = siteId
      if (siteName) this.site.name = siteName
    },
    showToast(type, message) {
      this.toast = { type, message, at: Date.now() }
      window.clearTimeout(this._toastTimer)
      this._toastTimer = window.setTimeout(() => {
        this.toast = null
      }, 2400)
    },
    clearToast() {
      this.toast = null
    },
    setCursor(line, column) {
      this.cursor = { line, column }
    },
    setActive(path) {
      this.activePath = path
      if (!this.openTabs.includes(path)) this.openTabs.push(path)
      let parent = dirname(path)
      while (parent && parent !== '/') {
        if (!this.expandedFolders.includes(parent)) this.expandedFolders.push(parent)
        parent = dirname(parent)
      }
    },
    closeTab(path) {
      this.openTabs = this.openTabs.filter((item) => item !== path)
      if (this.activePath === path) {
        this.activePath = this.openTabs[this.openTabs.length - 1] || '/index.html'
      }
    },
    toggleFolder(path) {
      if (this.expandedFolders.includes(path)) {
        this.expandedFolders = this.expandedFolders.filter((item) => item !== path)
      } else {
        this.expandedFolders = [...this.expandedFolders, path]
      }
    },
    isFolderExpanded(path) {
      return this.expandedFolders.includes(path)
    },
    updateFile(path, content) {
      const file = this.files.find((item) => item.path === path)
      if (!file) return
      file.content = content
      this.dirtyPaths = { ...this.dirtyPaths, [path]: true }
      file.dirty = true
      this.site.status = this.site.status === 'Live' ? 'Live' : 'Draft'
    },
    async loadSite(siteId) {
      this.error = ''
      const { data } = await api.get(`/projects/${siteId}`)
      const site = data?.item || data || {}
      this.site = {
        id: site.id || siteId,
        name: site.name || 'Untitled site',
        status: site.status || 'Draft',
        publicUrl: site.publicUrl || site.previewUrl || ''
      }
    },
    async loadWorkspace(siteId) {
      this.loading = true
      this.error = ''
      try {
        const { data } = await api.get(`/sites/${siteId}/workspace/files`)
        const items = Array.isArray(data?.items) ? data.items : []
        const folderItems = items.filter((item) => item.kind === 'folder').map((item) => ({
          path: buildPath(item.path),
          name: item.name || basename(item.path)
        }))
        const fileItems = items.filter((item) => item.kind !== 'folder')
        const files = []
        for (const item of fileItems) {
          const normalizedPath = buildPath(item.path)
          const response = await api.get(`/sites/${siteId}/workspace/files/content`, {
            params: { path: item.path }
          })
          files.push({
            id: normalizedPath,
            name: item.name || basename(normalizedPath),
            path: normalizedPath,
            type: item.fileType || 'txt',
            content: response.data?.item?.content || '',
            dirty: false
          })
        }
        this.folders = folderItems
        this.files = files.length ? files : [{
          id: '/index.html',
          name: 'index.html',
          path: '/index.html',
          type: 'html',
          content: '<!doctype html><html><head></head><body><main></main></body></html>',
          dirty: false
        }]
        this.dirtyPaths = {}
        this.validationErrors = []
        this.validationStatus = 'Not validated'
        this.publishStatus = this.site.status === 'Live' ? 'Live' : 'Draft only'
        this.openTabs = this.files.slice(0, 2).map((file) => file.path)
        if (!this.fileMap[this.activePath]) this.activePath = this.files[0].path
        await this.refreshPreview()
      } catch (error) {
        this.error = error.response?.data?.error?.message || 'The editor workspace could not be loaded.'
      } finally {
        this.loading = false
      }
    },
    validateName(name, isFolder = false) {
      const trimmed = String(name || '').trim()
      if (!trimmed) return 'Name is required'
      if (!SAFE_NAME.test(trimmed)) return 'Use letters, numbers, dashes, underscores, and dots only'
      if (!isFolder && !/\.[a-z0-9]+$/i.test(trimmed)) return 'A file extension is required'
      return ''
    },
    validate() {
      const errors = []
      const seen = new Set()
      if (!this.fileMap['/index.html']) {
        errors.push({ path: '/index.html', file: 'index.html', message: 'index.html is required' })
      }
      this.files.forEach((file) => {
        if (seen.has(file.path)) errors.push({ path: file.path, file: file.name, message: 'Duplicate file path detected' })
        seen.add(file.path)
        const nameError = this.validateName(file.name, false)
        if (nameError) errors.push({ path: file.path, file: file.name, message: nameError })
        if (!file.content.trim()) errors.push({ path: file.path, file: file.name, message: 'File content cannot be empty' })
        if (file.type === 'html') {
          const content = file.content.toLowerCase()
          if (!content.includes('<')) errors.push({ path: file.path, file: file.name, message: 'HTML file does not look like valid markup' })
          if (file.path === '/index.html' && !content.includes('<body')) errors.push({ path: file.path, file: file.name, message: 'index.html should include a body element' })
        }
      })
      this.validationErrors = errors
      this.validationStatus = errors.length ? `${errors.length} issues` : 'Clean'
      return errors
    },
    async saveFile(siteId, path) {
      const file = this.fileMap[path]
      if (!file) return
      this.saveLoading = true
      try {
        await api.put(`/sites/${siteId}/workspace/files/content`, {
          path: path.replace(/^\//, ''),
          content: file.content
        })
        this.dirtyPaths = { ...this.dirtyPaths, [path]: false }
        file.dirty = false
        this.lastSavedAt = nowIso()
        this.site.status = this.site.status === 'Live' ? 'Live' : 'Saved'
        this.showToast('success', `${file.name} saved`)
      } finally {
        this.saveLoading = false
      }
    },
    async saveActive(siteId) {
      if (!this.activeFile) return
      await this.saveFile(siteId, this.activeFile.path)
      await this.refreshPreview()
    },
    async saveAllDirty(siteId) {
      const dirtyPaths = Object.keys(this.dirtyPaths).filter((path) => this.dirtyPaths[path])
      for (const path of dirtyPaths) {
        await this.saveFile(siteId, path)
      }
    },
    async createFolder(siteId, parentPath, name) {
      const error = this.validateName(name, true)
      if (error) throw new Error(error)
      const nextPath = buildPath(parentPath, name)
      if (this.allFolders.some((folder) => folder.path === nextPath)) throw new Error('A folder with that name already exists')
      await api.post(`/sites/${siteId}/workspace/folders`, { path: nextPath.replace(/^\//, '') })
      await this.loadWorkspace(siteId)
      this.expandedFolders = [...new Set([...this.expandedFolders, nextPath])]
      this.showToast('success', `${name} created`)
    },
    async createFile(siteId, parentPath, name, type) {
      const error = this.validateName(name, false)
      if (error) throw new Error(error)
      const nextPath = buildPath(parentPath, name)
      if (this.fileMap[nextPath]) throw new Error('A file with that name already exists')
      await api.post(`/sites/${siteId}/workspace/files`, {
        path: nextPath.replace(/^\//, ''),
        fileType: type,
        content: defaultContentForType(type, name)
      })
      await this.loadWorkspace(siteId)
      this.setActive(nextPath)
      this.updateFile(nextPath, this.fileMap[nextPath]?.content || defaultContentForType(type, name))
      this.showToast('success', `${name} created`)
    },
    async renameEntry(siteId, node, nextName) {
      const isFolder = node.kind === 'folder'
      const error = this.validateName(nextName, isFolder)
      if (error) throw new Error(error)
      if (node.protected) throw new Error('This item cannot be renamed')
      const nextPath = buildPath(dirname(node.path), nextName)
      if (isFolder) {
        await api.put(`/sites/${siteId}/workspace/folders/rename`, {
          path: node.path.replace(/^\//, ''),
          newPath: nextPath.replace(/^\//, '')
        })
      } else {
        await api.put(`/sites/${siteId}/workspace/files/rename`, {
          path: node.path.replace(/^\//, ''),
          newPath: nextPath.replace(/^\//, '')
        })
      }
      await this.loadWorkspace(siteId)
      if (!isFolder) this.setActive(nextPath)
      this.showToast('success', `${node.name} renamed`)
    },
    async deleteEntry(siteId, node) {
      if (node.protected) throw new Error('This item cannot be deleted')
      if (node.kind === 'folder') {
        await api.delete(`/sites/${siteId}/workspace/folders`, {
          data: { path: node.path.replace(/^\//, '') }
        })
      } else {
        await api.delete(`/sites/${siteId}/workspace/files`, {
          data: { path: node.path.replace(/^\//, '') }
        })
      }
      await this.loadWorkspace(siteId)
      this.activePath = this.fileMap['/index.html'] ? '/index.html' : this.files[0]?.path || ''
      this.showToast('success', `${node.name} deleted`)
    },
    async duplicateEntry(siteId, node) {
      if (node.kind === 'folder') {
        const duplicateFolderPath = `${node.path}-copy`
        await this.createFolder(siteId, dirname(node.path), `${basename(node.path)}-copy`)
        const descendants = this.files.filter((file) => file.path.startsWith(node.path + '/'))
        for (const file of descendants) {
          const relativePath = file.path.slice(node.path.length + 1)
          const nextPath = buildPath(duplicateFolderPath, relativePath)
          await api.post(`/sites/${siteId}/workspace/files`, {
            path: nextPath.replace(/^\//, ''),
            fileType: file.type,
            content: file.content
          })
        }
        await this.loadWorkspace(siteId)
        this.showToast('success', `${node.name} duplicated`)
        return
      }
      await api.post(`/sites/${siteId}/workspace/files/duplicate`, { path: node.path.replace(/^\//, '') })
      await this.loadWorkspace(siteId)
      this.showToast('success', `${node.name} duplicated`)
    },
    async publish(siteId) {
      const errors = this.validate()
      if (errors.length) {
        this.showToast('error', 'Resolve validation issues before publishing')
        return { ok: false, errors }
      }
      this.publishLoading = true
      this.publishStatus = 'Publishing...'
      try {
        await this.saveAllDirty(siteId)
        const { data } = await api.post(`/sites/${siteId}/publish`)
        if (data?.success === false && Array.isArray(data?.errors)) {
          this.validationErrors = data.errors.map((error) => ({
            path: error.file ? `/${String(error.file).replace(/^\/+/, '')}` : this.activePath,
            file: error.file || basename(this.activePath),
            line: error.line,
            column: error.column,
            message: error.message
          }))
          this.validationStatus = `${this.validationErrors.length} issues`
          this.publishStatus = 'Publish blocked'
          this.showToast('error', 'Resolve template errors before publishing')
          return { ok: false, errors: this.validationErrors }
        }
        const url = data?.public_url || data?.item?.public_url || ''
        this.site.publicUrl = url
        this.site.status = 'Live'
        this.lastPublishedAt = nowIso()
        this.publishStatus = 'Live'
        this.publishResult = { title: 'Your site is live', url }
        this.validationErrors = []
        this.validationStatus = 'Clean'
        this.showToast('success', 'Site published successfully')
        return { ok: true, url }
      } catch (error) {
        const responseErrors = error?.response?.data?.errors
        if (Array.isArray(responseErrors) && responseErrors.length) {
          this.validationErrors = responseErrors.map((item) => ({
            path: item.file ? `/${String(item.file).replace(/^\/+/, '')}` : this.activePath,
            file: item.file || basename(this.activePath),
            line: item.line,
            column: item.column,
            message: item.message
          }))
          this.validationStatus = `${this.validationErrors.length} issues`
          this.publishStatus = 'Publish blocked'
          this.showToast('error', 'Resolve template errors before publishing')
          return { ok: false, errors: this.validationErrors }
        }
        throw error
      } finally {
        this.publishLoading = false
      }
    },
    async refreshPreview() {
      if (!this.site.id || !this.activePath) {
        this.previewHtml = ''
        return
      }
      this.previewLoading = true
      this.previewNonce += 1
      try {
        const response = await api.get(`/sites/${this.site.id}/preview`, {
          params: {
            path: String(this.activePath || '/index.html').replace(/^\//, '') || 'index.html',
            t: this.previewNonce
          },
          responseType: 'text',
          validateStatus: () => true
        })
        this.previewHtml = typeof response.data === 'string' ? response.data : ''
        if (response.status >= 400) {
          this.validationStatus = 'Preview error'
        }
      } catch (error) {
        this.previewHtml = '<!doctype html><html><body><main style="font-family:Segoe UI,Arial,sans-serif;padding:24px">Preview could not be loaded.</main></body></html>'
      } finally {
        this.previewLoading = false
      }
    }
  }
})
