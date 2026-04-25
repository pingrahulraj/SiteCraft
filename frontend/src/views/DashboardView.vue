<template>
  <main class="app-shell">
    <header class="topbar">
      <div class="topbar-brand">
        <span class="brand-mark brand-mark-sm"></span>
        <div>
          <strong>SiteCraft</strong>
          <span>Workspace</span>
        </div>
      </div>
      <div class="topbar-actions">
        <button class="topbar-button" @click="showModal = true">New project</button>
        <button class="topbar-button topbar-button-ghost" @click="signOut">Sign out</button>
      </div>
    </header>

    <div class="workspace">
      <aside class="sidebar">
        <div class="sidebar-section">
          <p class="sidebar-label">Workspace</p>
          <nav class="sidebar-nav">
            <a class="sidebar-link active" href="#">Overview</a>
            <a class="sidebar-link" href="#">Projects</a>
            <a class="sidebar-link" href="#">Templates</a>
            <a class="sidebar-link" href="#">Deployments</a>
          </nav>
        </div>

        <div class="sidebar-section sidebar-card">
          <p class="sidebar-label">Tenant</p>
          <strong>Acme Studio</strong>
          <p class="sidebar-copy">Multi-tenant workspace with seeded starter data.</p>
        </div>
      </aside>

      <section class="content">
        <div class="content-hero">
          <div class="content-hero-copy">
            <p class="eyebrow">Workspace overview</p>
            <h1>Build, preview, and publish websites from one clean control plane.</h1>
            <p class="lead">
              A Carbon-inspired workspace that keeps the core actions visible, organized, and fast to scan.
            </p>
          </div>
          <div class="hero-metrics">
            <div class="metric"><span>Sites</span><strong>{{ projects.length }}</strong></div>
            <div class="metric"><span>Published</span><strong>{{ publishedCount }}</strong></div>
            <div class="metric"><span>Drafts</span><strong>{{ draftCount }}</strong></div>
          </div>
        </div>

        <section class="project-list">
          <div v-if="loading" class="surface-card">
            <p class="eyebrow">Loading</p>
            <h2>Fetching projects</h2>
            <p class="lead">Loading the current workspace projects.</p>
          </div>
          <div v-else-if="loadError" class="surface-card">
            <p class="eyebrow">Fallback</p>
            <h2>Showing seed projects</h2>
            <p class="lead">{{ loadError }}</p>
          </div>
          <article v-for="project in projects" :key="project.id" class="project-card" role="button" tabindex="0" @click="openProject(project.id)" @keydown.enter="openProject(project.id)">
            <div class="project-thumb" :class="thumbClass(project.id)">
              <span>{{ project.name.slice(0, 2).toUpperCase() }}</span>
            </div>
            <div class="project-card-top">
              <div>
                <p class="eyebrow">{{ project.status }}</p>
                <h2>{{ project.name }}</h2>
              </div>
              <span class="project-pill">Open</span>
            </div>
            <p class="project-copy">{{ project.description || 'No description provided yet.' }}</p>
            <div class="project-meta">
              <span>{{ project.previewUrl }}</span>
              <span>Updated {{ project.createdAt }}</span>
            </div>
          </article>
        </section>
      </section>
    </div>

    <div v-if="showModal" class="modal-backdrop" @click.self="closeModal">
      <section class="modal-shell">
        <div class="modal-header">
          <div>
            <p class="eyebrow">New project</p>
            <h2>Create a site</h2>
          </div>
          <button class="icon-button" @click="closeModal">x</button>
        </div>
        <form class="modal-form" @submit.prevent="createProject">
          <label class="field">
            <span>Site name</span>
            <input v-model="form.name" class="input" type="text" placeholder="Marketing site" />
          </label>
          <label class="field">
            <span>Details</span>
            <textarea v-model="form.description" class="input textarea" rows="4" placeholder="Short description of this website build"></textarea>
          </label>
          <div class="modal-actions">
            <button type="button" class="button secondary" @click="closeModal">Cancel</button>
            <button type="submit" class="button" :disabled="saving">{{ saving ? 'Creating...' : 'Create site' }}</button>
          </div>
        </form>
      </section>
    </div>
  </main>
</template>

<script setup>
import { computed, onMounted, reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import { api } from '../services/api'

const router = useRouter()
const tenantSlug = 'acme-studio'
const projects = ref([])
const loading = ref(false)
const loadError = ref('')
const showModal = ref(false)
const saving = ref(false)
const form = reactive({ name: '', description: '' })

const fallbackProjects = () => ([
  { id: 1, name: 'Marketing site', description: 'Primary public-facing site with a clean landing experience.', status: 'published', previewUrl: 'http://localhost:8081/acme-studio/marketing', createdAt: 'seeded' },
  { id: 2, name: 'Builder console', description: 'Internal workspace for composing and managing HTML output.', status: 'draft', previewUrl: 'http://localhost:8081/acme-studio/console', createdAt: 'seeded' },
  { id: 3, name: 'Client landing page', description: 'Tenant-specific campaign landing page ready for review.', status: 'review', previewUrl: 'http://localhost:8081/acme-studio/client', createdAt: 'seeded' }
])

const loadProjects = async () => {
  loading.value = true
  loadError.value = ''
  try {
    const { data } = await api.get('/projects', { params: { tenantSlug } })
    projects.value = data.projects?.length ? data.projects : fallbackProjects()
  } catch {
    loadError.value = 'The live project list could not be loaded, so the seeded examples are shown instead.'
    projects.value = fallbackProjects()
  } finally {
    loading.value = false
  }
}

const publishedCount = computed(() => projects.value.filter((item) => item.status === 'published').length)
const draftCount = computed(() => projects.value.filter((item) => item.status !== 'published').length)
const thumbClass = (id) => `thumb-${(id % 4) + 1}`
const openProject = (id) => router.push(`/site/${id}`)
const closeModal = () => {
  showModal.value = false
  form.name = ''
  form.description = ''
}
const createProject = async () => {
  saving.value = true
  try {
    await api.post('/projects', { tenantSlug, name: form.name, description: form.description })
    await loadProjects()
    closeModal()
  } finally {
    saving.value = false
  }
}
const signOut = async () => {
  localStorage.removeItem('sitecraft_token')
  await router.push('/login')
}
onMounted(loadProjects)
</script>
