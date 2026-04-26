<template>
  <aside class="navigator-panel navigator-panel-pro">
    <div class="panel-head">
      <div>
        <p class="sidebar-label">Navigator</p>
        <h2>Workspace</h2>
      </div>
    </div>

    <div class="navigator-group">
      <div class="navigator-group-head">
        <span>Pages</span>
        <button class="link-button" @click="$emit('create-file', { parentPath: '/', type: 'html' })">Add page</button>
      </div>
      <button
        v-for="page in pages"
        :key="page.path"
        class="nav-item"
        :class="{ active: activePath === page.path, dirty: !!page.dirty }"
        @click="$emit('select-file', page.path)"
        @contextmenu.prevent="$emit('open-context', { event: $event, node: page })"
      >
        <span>{{ page.path === '/index.html' ? 'Home' : page.name }}</span>
        <span class="nav-path">{{ page.path }}</span>
      </button>
    </div>

    <div class="navigator-group">
      <div class="navigator-group-head">
        <span>Files</span>
        <div class="navigator-actions">
          <button class="link-button" @click="$emit('create-file', { parentPath: '/', type: 'css' })">CSS</button>
          <button class="link-button" @click="$emit('create-file', { parentPath: '/', type: 'js' })">JS</button>
          <button class="link-button" @click="$emit('create-folder', '/')">Folder</button>
        </div>
      </div>
      <div class="tree">
        <NavigatorNode
          v-for="node in tree"
          :key="node.path"
          :node="node"
          :active-path="activePath"
          :expanded-paths="expandedPaths"
          @toggle-folder="$emit('toggle-folder', $event)"
          @select-file="$emit('select-file', $event)"
          @open-context="$emit('open-context', $event)"
        />
      </div>
    </div>

    <div class="navigator-group">
      <div class="navigator-group-head">
        <span>Assets</span>
        <button class="link-button" @click="$emit('upload')">Upload</button>
      </div>
      <div class="assets-empty">
        <p>Assets are ready for upload and file linking.</p>
      </div>
    </div>
  </aside>
</template>

<script setup>
import NavigatorNode from './NavigatorNode.vue'

defineProps({
  pages: { type: Array, default: () => [] },
  tree: { type: Array, default: () => [] },
  activePath: { type: String, default: '' },
  expandedPaths: { type: Array, default: () => [] }
})

defineEmits(['select-file', 'toggle-folder', 'create-file', 'create-folder', 'open-context', 'upload'])
</script>
