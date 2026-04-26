<template>
  <div class="modal-backdrop" @click.self="$emit('close')">
    <div class="modal-shell publish-modal-shell">
      <p class="eyebrow">Publish result</p>
      <h2>Your site is live</h2>
      <p class="lead">{{ url }}</p>
      <div class="modal-actions">
        <button class="button secondary" @click="$emit('copy')">Copy Link</button>
        <button class="button secondary" @click="$emit('open')">Open Website</button>
        <button class="button" @click="$emit('close')">Continue Editing</button>
      </div>
    </div>
  </div>
</template>

<script setup>
defineProps({
  url: { type: String, default: '' }
})

defineEmits(['copy', 'open', 'close'])
</script>
