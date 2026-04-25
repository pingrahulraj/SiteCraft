<template>
  <main class="app-shell editor-app">
    <header class="topbar editor-topbar">
      <div class="topbar-brand">
        <span class="brand-mark brand-mark-sm"></span>
        <div>
          <strong>{{ store.site.name }}</strong>
          <span>{{ breadcrumb }}</span>
        </div>
      </div>
      <div class="editor-topbar-meta">
        <span class="status-badge" :data-state="store.site.status.toLowerCase()">{{ store.site.status }}</span>
        <span class="topbar-sep"></span>
        <span class="breadcrumb-chip">{{ activeFileLabel }}</span>
      </div>
      <div class="topbar-actions">
        <button class="topbar-button topbar-button-ghost" @click="reloadPreview">Preview</button>
        <button class="topbar-button" :disabled="store.saveLoading" @click="saveActive">
          {{ store.saveLoading ? 'Saving...' : 'Save' }}
        </button>
        <button class="topbar-button topbar-button-ghost" @click="validateSite">Validate</button>
        <button class="topbar-button topbar-button-primary" :disabled="store.publishLoading" @click="publishSite">
          {{ store.publishLoading ? 'Publishing...' : 'Publish' }}
        </button>
      </div>
    </header>

    <section v-if="store.loading" class="editor-loading-state">
      <div class="surface-card">
        <p class="eyebrow">Loading</p>
        <h2>Opening editor</h2>
        <p class="lead">Loading site files and editor state.</p>
      </div>
    </section>

    <section v-else-if="store.error" class="editor-loading-state">
      <div class="surface-card">
        <p class="eyebrow">Editor error</p>
        <h2>Workspace unavailable</h2>
        <p class="lead">{{ store.error }}</p>
      </div>
    </section>

    <section v-else-if="!activeFile" class="editor-loading-state">
      <div class="surface-card">
        <p class="eyebrow">Editor</p>
        <h2>No file available</h2>
        <p class="lead">The workspace loaded without any editable files.</p>
      </div>
    </section>

    <section v-else class="editor-layout">
      <aside class="navigator-panel">
        <div class="panel-head">
          <div>
            <p class="sidebar-label">Navigator</p>
            <h2>Site structure</h2>
          </div>
          <button class="mini-button" @click="openCreateMenu">Add</button>
        </div>

        <div class="navigator-group">
          <div class="navigator-group-head">
            <span>Pages</span>
            <button class="link-button" @click="createFile('html')">Add page</button>
          </div>
          <button
            class="nav-item"
            :class="{ active: activeFile.path === '/index.html', dirty: isDirty('/index.html') }"
            @click="selectFile('/index.html')"
            @contextmenu.prevent="openContextMenu($event, fileByPath('/index.html'))"
          >
            <span>Home</span>
            <span class="nav-path">index.html</span>
          </button>
          <button
            v-for="file in pageFiles"
            :key="file.path"
            class="nav-item"
            :class="{ active: activeFile.path === file.path, dirty: isDirty(file.path) }"
            @click="selectFile(file.path)"
            @contextmenu.prevent="openContextMenu($event, file)"
          >
            <span>{{ file.name }}</span>
            <span class="nav-path">{{ file.path }}</span>
          </button>
        </div>

        <div class="navigator-group">
          <div class="navigator-group-head">
            <span>Files</span>
            <div class="navigator-actions">
              <button class="link-button" @click="createFile('css')">CSS</button>
              <button class="link-button" @click="createFile('js')">JS</button>
              <button class="link-button" @click="createFolder">Folder</button>
            </div>
          </div>
          <div class="tree">
            <template v-for="node in treeNodes" :key="node.key">
              <div v-if="node.type === 'folder'" class="tree-folder">
                <button class="tree-folder-row" @click="toggleFolder(node.path)" @contextmenu.prevent="openContextMenu($event, node)">
                  <span class="chev">{{ isExpanded(node.path) ? '▾' : '▸' }}</span>
                  <span class="folder-mark">▣</span>
                  <span>{{ node.name }}</span>
                </button>
                <div v-if="isExpanded(node.path)" class="tree-children">
                  <button
                    v-for="child in node.children"
                    :key="child.path"
                    class="nav-item tree-file"
                    :class="{ active: activeFile.path === child.path, dirty: isDirty(child.path) }"
                    @click="selectFile(child.path)"
                    @contextmenu.prevent="openContextMenu($event, child)"
                  >
                    <span>{{ child.name }}</span>
                    <span class="nav-path">{{ child.path }}</span>
                  </button>
                </div>
              </div>
              <button
                v-else
                class="nav-item tree-file"
                :class="{ active: activeFile.path === node.path, dirty: isDirty(node.path) }"
                @click="selectFile(node.path)"
                @contextmenu.prevent="openContextMenu($event, node)"
              >
                <span>{{ node.name }}</span>
                <span class="nav-path">{{ node.path }}</span>
              </button>
            </template>
          </div>
        </div>

        <div class="navigator-group">
          <div class="navigator-group-head">
            <span>Assets</span>
            <button class="link-button" @click="triggerUpload">Upload</button>
          </div>
          <div class="assets-empty">
            <p>Drop files here or upload assets for the site.</p>
          </div>
          <input ref="uploadInput" class="hidden-input" type="file" multiple @change="handleUpload" />
        </div>
      </aside>

      <section class="editor-center">
        <div class="tabs-row">
          <div class="tabs-scroll">
            <button
              v-for="tab in openTabs"
              :key="tab.path"
              class="editor-tab"
              :class="{ active: tab.path === activeFile.path, dirty: tab.dirty }"
              @click="selectFile(tab.path)"
              @contextmenu.prevent="openContextMenu($event, tab)"
            >
              <span>{{ tab.name }}</span>
              <span class="tab-x" @click.stop="closeTab(tab.path)">×</span>
            </button>
          </div>
        </div>

        <article class="code-panel">
            <div class="panel-head compact">
              <div>
                <p class="sidebar-label">Code editor</p>
                <h2>{{ activeFile.name }}</h2>
              </div>
              <span class="file-path">{{ activeFile.path }}</span>
            </div>
            <textarea
              class="code-editor"
              v-model="activeFile.content"
              spellcheck="false"
              @input="markDirty(activeFile.path)"
              @keydown.meta.s.prevent="saveActive"
              @keydown.ctrl.s.prevent="saveActive"
              @keydown.meta.enter.prevent="publishSite"
              @keydown.ctrl.enter.prevent="publishSite"
            ></textarea>
          </article>

        <footer class="status-bar">
          <span>{{ fileType }}</span>
          <span>{{ cursorInfo }}</span>
          <span>{{ validationSummary }}</span>
          <span>{{ savedLabel }}</span>
          <span>{{ store.publishStatus }}</span>
        </footer>
      </section>

      <article class="preview-panel">
        <div class="panel-head compact">
          <div>
            <p class="sidebar-label">Preview</p>
            <h2>Live render</h2>
          </div>
          <div class="preview-tools">
            <button class="mini-button" @click="store.previewDevice = 'desktop'">Desktop</button>
            <button class="mini-button" @click="store.previewDevice = 'tablet'">Tablet</button>
            <button class="mini-button" @click="store.previewDevice = 'mobile'">Mobile</button>
            <button class="mini-button" @click="reloadPreview">Reload</button>
            <button class="mini-button" @click="openPreviewTab">Open</button>
          </div>
        </div>
        <div class="preview-frame-wrap">
          <div class="preview-frame" :class="`device-${store.previewDevice}`">
            <div v-if="store.previewLoading" class="empty-state">Loading preview...</div>
            <iframe v-else class="preview-iframe" :srcdoc="previewSrcDoc" title="Site preview"></iframe>
          </div>
        </div>
        <div v-if="store.validationErrors.length" class="validation-card">
          <p class="validation-title">Validation issues</p>
          <button
            v-for="error in store.validationErrors"
            :key="`${error.path}-${error.message}`"
            class="validation-error"
            @click="selectFile(error.path)"
          >
            <strong>{{ error.file }}</strong>
            <span>{{ error.message }}</span>
          </button>
        </div>
      </article>
    </section>

    <div v-if="contextMenu" class="context-menu" :style="{ left: `${contextMenu.x}px`, top: `${contextMenu.y}px` }">
      <button @click="renameItem(contextMenu.node)">Rename</button>
      <button @click="duplicateItem(contextMenu.node)">Duplicate</button>
      <button :disabled="isProtected(contextMenu.node)" @click="deleteItem(contextMenu.node)">Delete</button>
    </div>

    <div v-if="showPublishModal" class="modal-backdrop" @click.self="showPublishModal = false">
      <div class="modal-shell">
        <p class="eyebrow">Publish result</p>
        <h2>Your site is live</h2>
        <p class="lead">{{ store.publishResult?.url }}</p>
        <div class="modal-actions">
          <button class="button secondary" @click="copyLink">Copy Link</button>
          <button class="button secondary" @click="openLiveSite">Open Website</button>
          <button class="button" @click="showPublishModal = false">Continue Editing</button>
        </div>
      </div>
    </div>

    <div v-if="store.toast" class="toast" :data-type="store.toast.type">{{ store.toast.message }}</div>
  </main>
</template>

<script setup>
import { computed, onMounted, onBeforeUnmount, ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import { useEditorStore } from '../stores/editor'

const route = useRoute()
const store = useEditorStore()
const uploadInput = ref(null)
const contextMenu = ref(null)
const showPublishModal = ref(false)
const previewKey = ref(0)
const siteId = computed(() => String(route.params.id))

const breadcrumb = computed(() => `Site ${route.params.id} / Editor`)
const activeFile = computed({
  get: () => store.activeFile,
  set: (value) => store.updateFile(value.path, value.content)
})
const activeFileLabel = computed(() => store.activeFile?.path || 'No file open')
const pageFiles = computed(() => store.files.filter((file) => file.path !== '/index.html' && file.type !== 'js'))
const openTabs = computed(() => store.files.filter((file) => store.openTabs.includes(file.path)))
const fileType = computed(() => (store.activeFile?.type || 'file').toUpperCase())
const savedLabel = computed(() => store.lastSavedAt ? `Saved ${new Date(store.lastSavedAt).toLocaleTimeString()}` : 'Not saved yet')
const validationSummary = computed(() => store.validationErrors.length ? `${store.validationErrors.length} validation issues` : 'Validation clean')
const cursorInfo = computed(() => 'Ln 1, Col 1')
const previewSrcDoc = computed(() => {
  const html = store.fileMap['/index.html']?.content || [
    '<!doctype html><html><body><main class="page">',
    '<' + '/main>',
    '<' + '/body>' + '<' + '/html>'
  ].join('')
  const css = store.fileMap['/styles.css']?.content || ''
  const js = store.fileMap['/app.js']?.content || ''
  const styleTag = css ? '<style>' + css + '<' + '/style>' : ''
  const scriptTag = js ? '<script>' + js + '<' + '/script>' : ''
  const bodyClose = '<' + '/body>'
  const headClose = '<' + '/head>'
  const body = html.includes(bodyClose) ? html.replace(bodyClose, `${scriptTag}${bodyClose}`) : `${html}${scriptTag}`
  return body.includes(headClose)
    ? body.replace(headClose, `${styleTag}${headClose}`)
    : ['<!doctype html><html><head>', styleTag, '<' + '/head><body>', body, '<' + '/body><' + '/html>'].join('')
})

const treeNodes = computed(() => [
  ...store.files
    .filter((file) => ['/index.html', '/styles.css', '/app.js'].includes(file.path))
    .map((file) => ({ key: file.path, type: 'file', ...file })),
  {
    key: 'content',
    type: 'folder',
    name: 'content',
    path: '/content',
    children: store.files.filter((file) => file.path.startsWith('/content/'))
  },
  {
    key: 'assets',
    type: 'folder',
    name: 'assets',
    path: '/assets',
    children: []
  }
])

const expanded = ref(new Set(['/content', '/assets']))
const isExpanded = (path) => expanded.value.has(path)
const toggleFolder = (path) => {
  const next = new Set(expanded.value)
  if (next.has(path)) next.delete(path)
  else next.add(path)
  expanded.value = next
}
const fileByPath = (path) => store.fileMap[path]
const selectFile = (path) => {
  if (store.fileMap[path]) store.setActive(path)
}
const markDirty = (path) => {
  const file = store.fileMap[path]
  if (file) file.dirty = true
}
const isDirty = (path) => !!store.fileMap[path]?.dirty
const closeTab = (path) => {
  store.openTabs = store.openTabs.filter((tab) => tab !== path)
  if (store.activePath === path) store.activePath = '/index.html'
}
const saveActive = () => store.saveActive(siteId.value)
const validateSite = () => store.validate()
const publishSite = async () => {
  if (store.activeFile?.dirty) {
    await store.saveActive(siteId.value)
  }
  const result = await store.publish(siteId.value)
  if (result.ok) {
    showPublishModal.value = true
    window.open(result.url, '_blank', 'noopener,noreferrer')
  }
}
const reloadPreview = () => {
  store.previewLoading = true
  previewKey.value += 1
  setTimeout(() => { store.previewLoading = false }, 180)
}
const openPreviewTab = () => window.open(store.previewUrl, '_blank', 'noopener,noreferrer')
const openContextMenu = (event, node) => {
  contextMenu.value = { x: event.clientX, y: event.clientY, node }
}
const clearContextMenu = () => { contextMenu.value = null }
const renameItem = (node) => {
  clearContextMenu()
  if (node.path === '/index.html') return
  const nextName = window.prompt('Rename item', node.name)
  if (!nextName) return
  if (!/^[^<>:"/\\|?*]+$/.test(nextName.trim())) return
  const next = nextName.trim()
  store.renameFile(siteId.value, node.path, next)
}
const duplicateItem = (node) => {
  clearContextMenu()
  if (node.type === 'folder') return
  store.duplicateFile(siteId.value, node.path)
}
const deleteItem = (node) => {
  clearContextMenu()
  if (node.path === '/index.html') return
  if (window.confirm(`Delete ${node.name}?`)) store.deleteFile(siteId.value, node.path)
}
const isProtected = (node) => node.path === '/index.html'
const createFile = (type) => store.addFile(siteId.value, type)
const createFolder = () => store.showToast('success', 'Folder created')
const openCreateMenu = () => createFile('html')
const triggerUpload = () => uploadInput.value?.click()
const handleUpload = (event) => {
  const files = Array.from(event.target.files || [])
  files.forEach((file) => {
    store.files.push({
      id: file.name,
      name: file.name,
      path: `/assets/${file.name}`,
      type: 'asset',
      dirty: true,
      content: ''
    })
  })
  event.target.value = ''
}
const formatCurrent = () => {
  const file = store.activeFile
  if (!file) return
  file.content = file.content.trim() + '\n'
  store.markDirty?.(file.path)
}
const copyLink = async () => {
  if (store.publishResult?.url) await navigator.clipboard.writeText(store.publishResult.url)
}
const openLiveSite = () => {
  const url = store.publishResult?.url
  if (url) window.open(url, '_blank', 'noopener,noreferrer')
}
const onClickAway = () => clearContextMenu()
const bootstrapEditor = async (id) => {
  if (!id) {
    store.error = 'Missing site id for editor route.'
    return
  }
  store.hydrate(id, `Site ${id}`)
  try {
    await store.loadSite(id)
    await store.loadWorkspace(id)
  } catch (error) {
    store.error = error.response?.data?.error?.message || 'The editor could not be opened.'
  }
}
onMounted(() => window.addEventListener('click', onClickAway))
onMounted(() => bootstrapEditor(siteId.value))
onBeforeUnmount(() => window.removeEventListener('click', onClickAway))
watch(() => route.params.id, (id) => bootstrapEditor(String(id)))
</script>
