<template>
  <div class="context-menu" :style="{ left: `${x}px`, top: `${y}px` }">
    <button @click="$emit('rename')">Rename</button>
    <button @click="$emit('duplicate')">Duplicate</button>
    <button @click="$emit('create-file')">New file</button>
    <button @click="$emit('create-folder')">New folder</button>
    <button :disabled="disabledDelete" @click="$emit('delete')">Delete</button>
  </div>
</template>

<script setup>
defineProps({
  x: { type: Number, required: true },
  y: { type: Number, required: true },
  disabledDelete: { type: Boolean, default: false }
})

defineEmits(['rename', 'duplicate', 'create-file', 'create-folder', 'delete'])
</script>
