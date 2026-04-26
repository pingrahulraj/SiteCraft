<template>
  <main class="app-shell editor-app">
    <EditorTopBar
      :site-name="store.site.name"
      :breadcrumb="breadcrumb"
      :file-label="activeFileLabel"
      :status="statusLabel"
      :status-tone="statusTone"
      :save-loading="store.saveLoading"
      :publish-loading="store.publishLoading"
      @preview="reloadPreview"
      @validate="validateSite"
      @save="saveActive"
      @publish="publishSite"
    />

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

    <section v-else class="editor-layout editor-layout-pro" :class="{ 'preview-collapsed': previewCollapsed }">
      <NavigatorPanel
        :pages="pageFiles"
        :tree="tree"
        :active-path="store.activePath"
        :expanded-paths="store.expandedFolders"
        @select-file="selectFile"
        @toggle-folder="store.toggleFolder"
        @create-file="promptCreateFile"
        @create-folder="promptCreateFolder"
        @open-context="openContextMenu"
        @upload="triggerUpload"
      />

      <section class="editor-center">
        <CodeEditorPanel
          :tabs="openTabs"
          :active-path="store.activePath"
          :active-file="activeFile"
          :cursor-label="cursorLabel"
          :validation-label="validationSummary"
          :saved-label="savedLabel"
          :publish-label="store.publishStatus"
          @select-tab="selectFile"
          @close-tab="store.closeTab"
          @change="onFileChange"
          @cursor-change="onCursorChange"
          @format="formatCurrent"
          @save="saveActive"
          @publish="publishSite"
          @open-context="openContextMenu"
        />
      </section>

      <PreviewPanel
        :device="store.previewDevice"
        :loading="store.previewLoading"
        :srcdoc="store.previewHtml"
        :nonce="store.previewNonce"
        :collapsed="previewCollapsed"
        :errors="store.validationErrors"
        @set-device="store.previewDevice = $event"
        @toggle-collapsed="previewCollapsed = !previewCollapsed"
        @reload="reloadPreview"
        @open-preview="openPreviewTab"
        @open-file="selectFile"
      />
    </section>

    <FileContextMenu
      v-if="contextMenu"
      :x="contextMenu.x"
      :y="contextMenu.y"
      :disabled-delete="!!contextMenu.node?.protected"
      @rename="promptRename"
      @duplicate="duplicateNode"
      @create-file="promptCreateFile({ parentPath: contextTargetParent, type: 'html' })"
      @create-folder="promptCreateFolder(contextTargetParent)"
      @delete="confirmDelete"
    />

    <ConfirmDialog
      v-if="pendingDelete"
      title="Delete item"
      :message="`Delete ${pendingDelete.name}? This cannot be undone.`"
      @confirm="deleteNode"
      @cancel="pendingDelete = null"
    />

    <PublishSuccessModal
      v-if="showPublishModal"
      :url="store.publishResult?.url || ''"
      @copy="copyLink"
      @open="openLiveSite"
      @close="showPublishModal = false"
    />

    <div v-if="store.toast" class="toast" :data-type="store.toast.type">{{ store.toast.message }}</div>

    <input ref="uploadInput" class="hidden-input" type="file" multiple @change="handleUpload" />
  </main>
</template>

<script setup>
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import { useEditorStore } from '../stores/editor'
import EditorTopBar from '../components/editor/EditorTopBar.vue'
import NavigatorPanel from '../components/editor/NavigatorPanel.vue'
import CodeEditorPanel from '../components/editor/CodeEditorPanel.vue'
import PreviewPanel from '../components/editor/PreviewPanel.vue'
import FileContextMenu from '../components/editor/FileContextMenu.vue'
import PublishSuccessModal from '../components/editor/PublishSuccessModal.vue'
import ConfirmDialog from '../components/editor/ConfirmDialog.vue'

const route = useRoute()
const store = useEditorStore()
const uploadInput = ref(null)
const contextMenu = ref(null)
const showPublishModal = ref(false)
const pendingDelete = ref(null)
const previewCollapsed = ref(false)

const siteId = computed(() => String(route.params.id || ''))
const activeFile = computed(() => store.activeFile)
const pageFiles = computed(() => store.pageFiles)
const tree = computed(() => store.tree)
const openTabs = computed(() => store.openTabs.map((path) => store.fileMap[path]).filter(Boolean).map((file) => ({
  ...file,
  dirty: !!store.dirtyPaths[file.path]
})))
const breadcrumb = computed(() => `Site ${siteId.value} / Editor`)
const activeFileLabel = computed(() => activeFile.value?.path || 'No file open')
const statusLabel = computed(() => store.site.status || 'Draft')
const statusTone = computed(() => {
  const normalized = String(store.site.status || '').toLowerCase()
  if (normalized === 'live') return 'live'
  if (normalized === 'saved') return 'saved'
  return 'draft'
})
const cursorLabel = computed(() => `Ln ${store.cursor.line}, Col ${store.cursor.column}`)
const validationSummary = computed(() => store.validationErrors.length ? `${store.validationErrors.length} issues` : store.validationStatus)
const savedLabel = computed(() => store.lastSavedAt ? `Saved ${new Date(store.lastSavedAt).toLocaleTimeString()}` : 'Not saved yet')
const contextTargetParent = computed(() => {
  const node = contextMenu.value?.node
  if (!node) return '/'
  return node.kind === 'folder' ? node.path : node.path.split('/').slice(0, -1).join('/') || '/'
})

const bootstrapEditor = async () => {
  if (!siteId.value) {
    store.error = 'Missing site id for editor route.'
    return
  }
  store.hydrate(siteId.value, localStorage.getItem('sitecraft_active_site_name') || `Site ${siteId.value}`)
  try {
    await store.loadSite(siteId.value)
    await store.loadWorkspace(siteId.value)
  } catch (error) {
    store.error = error.response?.data?.error?.message || error.message || 'The editor could not be opened.'
  }
}

const selectFile = (path) => {
  if (store.fileMap[path]) store.setActive(path)
  clearContextMenu()
}

const onFileChange = ({ path, content }) => store.updateFile(path, content)
const onCursorChange = ({ line, column }) => store.setCursor(line, column)
const saveActive = async () => {
  await store.saveActive(siteId.value)
}
const validateSite = () => store.validate()
const publishSite = async () => {
  const result = await store.publish(siteId.value)
  if (result?.ok) {
    showPublishModal.value = true
  }
}
const reloadPreview = () => store.refreshPreview()
const openPreviewTab = () => {
  if (!store.previewHtml) return
  const blob = new Blob([store.previewHtml], { type: 'text/html' })
  const url = URL.createObjectURL(blob)
  window.open(url, '_blank', 'noopener,noreferrer')
  window.setTimeout(() => URL.revokeObjectURL(url), 60000)
}
const openLiveSite = () => {
  const url = store.publishResult?.url
  if (url) window.open(url, '_blank', 'noopener,noreferrer')
}
const copyLink = async () => {
  const url = store.publishResult?.url
  if (url) {
    await navigator.clipboard.writeText(url)
    store.showToast('success', 'Live URL copied')
  }
}
const clearContextMenu = () => {
  contextMenu.value = null
}
const openContextMenu = ({ event, node }) => {
  contextMenu.value = { x: event.clientX, y: event.clientY, node }
}
const promptName = (title, value = '') => window.prompt(title, value)
const promptCreateFile = async ({ parentPath = '/', type = 'html' }) => {
  clearContextMenu()
  const ext = type === 'css' ? 'css' : type === 'js' ? 'js' : 'html'
  const suggestion = type === 'html' ? 'new-page.html' : type === 'css' ? 'styles.css' : 'app.js'
  const name = promptName(`Create ${ext.toUpperCase()} file`, suggestion)
  if (!name) return
  try {
    await store.createFile(siteId.value, parentPath, name.trim(), ext)
  } catch (error) {
    store.showToast('error', error.message)
  }
}
const promptCreateFolder = async (parentPath = '/') => {
  clearContextMenu()
  const name = promptName('Create folder', 'new-folder')
  if (!name) return
  try {
    await store.createFolder(siteId.value, parentPath, name.trim())
  } catch (error) {
    store.showToast('error', error.message)
  }
}
const promptRename = async () => {
  const node = contextMenu.value?.node
  if (!node) return
  clearContextMenu()
  const nextName = promptName('Rename item', node.name)
  if (!nextName) return
  try {
    await store.renameEntry(siteId.value, node, nextName.trim())
  } catch (error) {
    store.showToast('error', error.message)
  }
}
const duplicateNode = async () => {
  const node = contextMenu.value?.node
  if (!node) return
  clearContextMenu()
  try {
    await store.duplicateEntry(siteId.value, node)
  } catch (error) {
    store.showToast('error', error.message)
  }
}
const confirmDelete = () => {
  const node = contextMenu.value?.node
  if (!node) return
  pendingDelete.value = node
  clearContextMenu()
}
const deleteNode = async () => {
  const node = pendingDelete.value
  if (!node) return
  pendingDelete.value = null
  try {
    await store.deleteEntry(siteId.value, node)
  } catch (error) {
    store.showToast('error', error.message)
  }
}
const formatCurrent = () => {
  if (!activeFile.value) return
  const trimmed = activeFile.value.content.replace(/\s+$/g, '')
  store.updateFile(activeFile.value.path, `${trimmed}\n`)
}
const triggerUpload = () => uploadInput.value?.click()
const handleUpload = (event) => {
  const files = Array.from(event.target.files || [])
  if (files.length) {
    store.showToast('success', `${files.length} asset placeholder${files.length > 1 ? 's' : ''} queued`)
  }
  event.target.value = ''
}
const onClickAway = () => clearContextMenu()

onMounted(() => {
  document.body.style.overflow = 'hidden'
  window.addEventListener('click', onClickAway)
  bootstrapEditor()
})
onBeforeUnmount(() => {
  document.body.style.overflow = ''
  window.removeEventListener('click', onClickAway)
})
watch(() => route.params.id, bootstrapEditor)
</script>
