<template>
  <footer class="status-bar status-bar-pro">
    <span>{{ fileType }}</span>
    <span>{{ cursorLabel }}</span>
    <span>{{ validationLabel }}</span>
    <span>{{ savedLabel }}</span>
    <span>{{ publishLabel }}</span>
  </footer>
</template>

<script setup>
defineProps({
  fileType: { type: String, default: 'FILE' },
  cursorLabel: { type: String, default: 'Ln 1, Col 1' },
  validationLabel: { type: String, default: 'Not validated' },
  savedLabel: { type: String, default: '' },
  publishLabel: { type: String, default: '' }
})
</script>
