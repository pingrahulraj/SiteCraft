<template>
  <div class="modal-backdrop" @click.self="$emit('cancel')">
    <div class="modal-shell confirm-modal-shell">
      <p class="eyebrow">Confirm action</p>
      <h2>{{ title }}</h2>
      <p class="lead">{{ message }}</p>
      <div class="modal-actions">
        <button class="button secondary" @click="$emit('cancel')">Cancel</button>
        <button class="button" @click="$emit('confirm')">Continue</button>
      </div>
    </div>
  </div>
</template>

<script setup>
defineProps({
  title: { type: String, default: 'Confirm action' },
  message: { type: String, default: '' }
})

defineEmits(['confirm', 'cancel'])
</script>
