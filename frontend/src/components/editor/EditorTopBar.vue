<template>
  <header class="topbar editor-topbar editor-topbar-pro">
    <div class="topbar-brand">
      <span class="brand-mark brand-mark-sm"></span>
      <div>
        <strong>{{ siteName }}</strong>
        <span>{{ breadcrumb }}</span>
      </div>
    </div>

    <div class="editor-topbar-meta">
      <span class="status-badge" :data-state="statusTone">{{ status }}</span>
      <span class="topbar-sep"></span>
      <span class="breadcrumb-chip">{{ fileLabel }}</span>
    </div>

    <div class="topbar-actions">
      <button class="topbar-button topbar-button-ghost" @click="$emit('preview')">Preview</button>
      <button class="topbar-button topbar-button-ghost" @click="$emit('validate')">Validate</button>
      <button class="topbar-button" :disabled="saveLoading" @click="$emit('save')">{{ saveLoading ? 'Saving...' : 'Save' }}</button>
      <button class="topbar-button topbar-button-primary" :disabled="publishLoading" @click="$emit('publish')">{{ publishLoading ? 'Publishing...' : 'Publish' }}</button>
    </div>
  </header>
</template>

<script setup>
defineProps({
  siteName: { type: String, default: 'Untitled site' },
  breadcrumb: { type: String, default: '' },
  fileLabel: { type: String, default: '' },
  status: { type: String, default: 'Draft' },
  statusTone: { type: String, default: 'saved' },
  saveLoading: { type: Boolean, default: false },
  publishLoading: { type: Boolean, default: false }
})

defineEmits(['preview', 'validate', 'save', 'publish'])
</script>
