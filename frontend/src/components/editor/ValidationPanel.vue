<template>
  <div v-if="errors.length" class="validation-card validation-card-pro">
    <div class="validation-card-head">
      <p class="validation-title">Validation issues</p>
      <span>{{ errors.length }} total</span>
    </div>
    <button
      v-for="error in errors"
      :key="`${error.path}-${error.message}`"
      class="validation-error"
      @click="$emit('open-file', error.path)"
    >
      <strong>{{ error.file }}</strong>
      <span>{{ error.message }}</span>
    </button>
  </div>
</template>

<script setup>
defineProps({
  errors: { type: Array, default: () => [] }
})

defineEmits(['open-file'])
</script>
