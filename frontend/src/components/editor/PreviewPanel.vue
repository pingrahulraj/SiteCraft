<template>
  <article class="preview-panel preview-panel-pro" :class="{ 'is-collapsed': collapsed }">
    <div class="panel-head compact">
      <div>
        <p class="sidebar-label">Preview</p>
        <h2>{{ collapsed ? 'Preview hidden' : 'Live render' }}</h2>
      </div>
      <div class="preview-tools">
        <button class="mini-button" :class="{ active: device === 'desktop' }" @click="$emit('set-device', 'desktop')">Desktop</button>
        <button class="mini-button" :class="{ active: device === 'tablet' }" @click="$emit('set-device', 'tablet')">Tablet</button>
        <button class="mini-button" :class="{ active: device === 'mobile' }" @click="$emit('set-device', 'mobile')">Mobile</button>
        <button class="mini-button" @click="$emit('toggle-collapsed')">{{ collapsed ? 'Expand' : 'Collapse' }}</button>
        <button class="mini-button" @click="$emit('reload')">Reload</button>
        <button class="mini-button" @click="$emit('open-preview')">Open</button>
      </div>
    </div>

    <div v-if="!collapsed" class="preview-frame-wrap">
      <div class="preview-frame" :class="`device-${device}`">
        <div v-if="loading" class="empty-state">Loading preview...</div>
        <div v-else-if="!srcdoc" class="empty-state">No preview available yet.</div>
        <iframe v-else :key="nonce" class="preview-iframe" :srcdoc="srcdoc" title="Site preview"></iframe>
      </div>
    </div>

    <ValidationPanel v-if="!collapsed" :errors="errors" @open-file="$emit('open-file', $event)" />
  </article>
</template>

<script setup>
import ValidationPanel from './ValidationPanel.vue'

defineProps({
  device: { type: String, default: 'desktop' },
  loading: { type: Boolean, default: false },
  srcdoc: { type: String, default: '' },
  nonce: { type: Number, default: 0 },
  collapsed: { type: Boolean, default: false },
  errors: { type: Array, default: () => [] }
})

defineEmits(['set-device', 'toggle-collapsed', 'reload', 'open-preview', 'open-file'])
</script>
