<template>
  <section class="editor-center">
    <div class="tabs-row">
      <div class="tabs-scroll">
        <button
          v-for="tab in tabs"
          :key="tab.path"
          class="editor-tab"
          :class="{ active: tab.path === activePath, dirty: !!tab.dirty }"
          @click="$emit('select-tab', tab.path)"
          @contextmenu.prevent="$emit('open-context', { event: $event, node: tab })"
        >
          <span>{{ tab.name }}</span>
          <span class="tab-x" @click.stop="$emit('close-tab', tab.path)">×</span>
        </button>
      </div>
      <div class="tabs-tools">
        <button class="mini-button" @click="$emit('format')">Format</button>
      </div>
    </div>

    <article class="code-panel code-panel-pro">
      <div class="panel-head compact">
        <div>
          <p class="sidebar-label">Code editor</p>
          <h2>{{ activeFile?.name || 'No file selected' }}</h2>
        </div>
        <span class="file-path">{{ activeFile?.path || '' }}</span>
      </div>

      <div v-if="activeFile" class="monaco-host" ref="editorHost"></div>
      <div v-else class="empty-state">Select a file to start editing.</div>
    </article>

    <BottomStatusBar
      :file-type="fileType"
      :cursor-label="cursorLabel"
      :validation-label="validationLabel"
      :saved-label="savedLabel"
      :publish-label="publishLabel"
    />
  </section>
</template>

<script setup>
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import BottomStatusBar from './BottomStatusBar.vue'

const props = defineProps({
  tabs: { type: Array, default: () => [] },
  activePath: { type: String, default: '' },
  activeFile: { type: Object, default: null },
  validationLabel: { type: String, default: 'Not validated' },
  savedLabel: { type: String, default: '' },
  publishLabel: { type: String, default: '' },
  cursorLabel: { type: String, default: 'Ln 1, Col 1' }
})

const emit = defineEmits(['select-tab', 'close-tab', 'change', 'cursor-change', 'format', 'save', 'publish', 'open-context'])

const editorHost = ref(null)
let monaco = null
let editor = null
let model = null
let suppressChange = false

const fileType = computed(() => String(props.activeFile?.type || 'file').toUpperCase())
const language = computed(() => {
  if (props.activeFile?.type === 'css') return 'css'
  if (props.activeFile?.type === 'js') return 'javascript'
  return 'html'
})

const ensureEditor = async () => {
  if (!editorHost.value) return
  if (!monaco) {
    monaco = await import('monaco-editor')
  }
  if (!editor) {
    editor = monaco.editor.create(editorHost.value, {
      value: '',
      language: language.value,
      theme: 'vs',
      automaticLayout: true,
      minimap: { enabled: false },
      fontSize: 13,
      lineHeight: 20,
      scrollBeyondLastLine: false,
      wordWrap: 'on',
      tabSize: 2,
      padding: { top: 12, bottom: 12 },
      glyphMargin: false
    })
    editor.onDidChangeModelContent(() => {
      if (suppressChange || !props.activeFile) return
      emit('change', { path: props.activeFile.path, content: editor.getValue() })
    })
    editor.onDidChangeCursorPosition((event) => {
      emit('cursor-change', {
        line: event.position.lineNumber,
        column: event.position.column
      })
    })
    editor.addCommand(monaco.KeyMod.CtrlCmd | monaco.KeyCode.KeyS, () => emit('save'))
    editor.addCommand(monaco.KeyMod.CtrlCmd | monaco.KeyCode.Enter, () => emit('publish'))
  }
  const nextValue = props.activeFile?.content || ''
  if (!model) {
    model = monaco.editor.createModel(nextValue, language.value)
    editor.setModel(model)
  } else {
    suppressChange = true
    monaco.editor.setModelLanguage(model, language.value)
    if (model.getValue() !== nextValue) model.setValue(nextValue)
    suppressChange = false
  }
}

watch(() => props.activeFile?.path, async () => {
  await nextTick()
  await ensureEditor()
})

watch(() => props.activeFile?.content, (value) => {
  if (!model || suppressChange) return
  const nextValue = value || ''
  if (model.getValue() === nextValue) return
  suppressChange = true
  model.setValue(nextValue)
  suppressChange = false
})

onMounted(async () => {
  await nextTick()
  await ensureEditor()
})

onBeforeUnmount(() => {
  model?.dispose()
  editor?.dispose()
})
</script>
