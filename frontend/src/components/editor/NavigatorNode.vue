<template>
  <div v-if="node.kind === 'folder'" class="tree-folder">
    <button class="tree-folder-row" @click="$emit('toggle-folder', node.path)" @contextmenu.prevent="$emit('open-context', { event: $event, node })">
      <span class="chev">{{ expanded ? '▾' : '▸' }}</span>
      <span class="folder-mark">▣</span>
      <span>{{ node.name }}</span>
    </button>
    <div v-if="expanded" class="tree-children">
      <NavigatorNode
        v-for="child in node.children"
        :key="child.path"
        :node="child"
        :active-path="activePath"
        :expanded-paths="expandedPaths"
        @toggle-folder="$emit('toggle-folder', $event)"
        @select-file="$emit('select-file', $event)"
        @open-context="$emit('open-context', $event)"
      />
    </div>
  </div>
  <button
    v-else
    class="nav-item tree-file"
    :class="{ active: activePath === node.path, dirty: !!node.dirty }"
    @click="$emit('select-file', node.path)"
    @contextmenu.prevent="$emit('open-context', { event: $event, node })"
  >
    <span>{{ node.name }}</span>
    <span class="nav-path">{{ node.path }}</span>
  </button>
</template>

<script setup>
import { computed } from 'vue'

const props = defineProps({
  node: { type: Object, required: true },
  activePath: { type: String, default: '' },
  expandedPaths: { type: Array, default: () => [] }
})

defineEmits(['toggle-folder', 'select-file', 'open-context'])

const expanded = computed(() => props.expandedPaths.includes(props.node.path))
</script>
