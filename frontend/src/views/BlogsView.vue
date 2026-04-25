<template>
  <main class="app-shell">
    <header class="topbar">
      <div class="topbar-brand">
        <span class="brand-mark brand-mark-sm"></span>
        <div>
          <strong>{{ siteName }}</strong>
          <span>Blogs</span>
        </div>
      </div>
      <div class="topbar-actions">
        <button class="topbar-button topbar-button-ghost" @click="backToSite">Back to Site</button>
        <button v-if="!isEditorRoute" class="topbar-button" @click="openNewPost">Add New Post</button>
      </div>
    </header>

    <div class="workspace">
      <aside class="sidebar sidebar-site">
        <div class="sidebar-section">
          <p class="sidebar-label">Site navigation</p>
          <nav class="sidebar-nav">
            <button
              v-for="item in siteSections"
              :key="item.key"
              class="sidebar-link"
              :class="{ active: item.key === 'blogs' }"
              @click="handleSiteSectionClick(item.key)"
            >
              {{ item.label }}
            </button>
          </nav>
        </div>

        <div class="sidebar-section sidebar-card">
          <p class="sidebar-label">Project</p>
          <strong>{{ siteName }}</strong>
          <p class="sidebar-copy">Use the site navigation to move between dashboard modules without losing the active site context.</p>
        </div>
      </aside>

      <section class="content blog-cms-shell">
        <div v-if="!resolvedSiteId" class="surface-card">
          <p class="eyebrow">Site required</p>
          <h2>Select a site first</h2>
          <p class="lead">Open Blogs from a site dashboard so the CMS can load the right publishing workspace.</p>
        </div>

        <template v-else-if="isEditorRoute">
          <div class="blog-editor-shell">
            <div class="blog-editor-topbar">
              <button class="topbar-button topbar-button-ghost" @click="goTo('/blogs/posts')">Back to Posts</button>
              <div class="blog-editor-actions">
                <button class="topbar-button topbar-button-ghost" @click="saveDraft">Save Draft</button>
                <button class="topbar-button topbar-button-ghost" @click="previewSite">Preview</button>
                <button class="topbar-button" @click="publishPost">{{ editorForm.id ? 'Update' : 'Publish' }}</button>
              </div>
            </div>

            <div class="blog-editor-layout">
              <section class="surface-card blog-editor-main">
                <label class="field">
                  <span>Title</span>
                  <input v-model="editorForm.title" class="input blog-title-input" placeholder="Add post title" />
                </label>
                <div class="permalink-row">
                  <strong>Permalink</strong>
                  <span>/{{ editorForm.slug || slugPreview }}</span>
                </div>
                <label class="field">
                  <span>Content</span>
                  <textarea v-model="editorForm.content" class="input textarea blog-content-editor" rows="18" placeholder="Write your post content here. HTML is supported."></textarea>
                </label>
              </section>

              <aside class="surface-card blog-editor-sidebar">
                <div class="blog-settings-group">
                  <p class="sidebar-label">Status & Visibility</p>
                  <label class="field"><span>Status</span><select v-model="editorForm.status" class="input"><option>draft</option><option>published</option><option>scheduled</option><option>private</option></select></label>
                  <label class="field"><span>Visibility</span><select v-model="editorForm.visibility" class="input"><option>public</option><option>members</option><option>paid</option></select></label>
                </div>
                <div class="blog-settings-group">
                  <p class="sidebar-label">Publishing</p>
                  <label class="field"><span>Publish date</span><input v-model="editorForm.scheduledAt" class="input" type="datetime-local" /></label>
                  <label class="field"><span>Author</span><select v-model="editorForm.author" class="input"><option value="">Select author</option><option v-for="author in blogs.authors" :key="author.id" :value="author.name">{{ author.name }}</option></select></label>
                </div>
                <div class="blog-settings-group">
                  <p class="sidebar-label">Organization</p>
                  <label class="field"><span>Categories</span><input v-model="editorCategories" class="input" placeholder="Announcements, Product" /></label>
                  <label class="field"><span>Tags</span><input v-model="editorForm.tags" class="input" placeholder="launch, update" /></label>
                  <label class="field"><span>Featured image</span><input v-model="editorForm.featuredImage" class="input" placeholder="https://example.com/image.jpg" /></label>
                </div>
                <div class="blog-settings-group">
                  <p class="sidebar-label">Meta</p>
                  <label class="field"><span>Excerpt</span><textarea v-model="editorForm.excerpt" class="input textarea" rows="4"></textarea></label>
                  <label class="field"><span>SEO title</span><input v-model="editorForm.seoTitle" class="input" /></label>
                  <label class="field"><span>SEO description</span><textarea v-model="editorForm.seoDescription" class="input textarea" rows="3"></textarea></label>
                </div>
                <p v-if="blogs.feedback" class="blog-feedback">{{ blogs.feedback }}</p>
                <p v-if="blogs.error" class="error">{{ blogs.error }}</p>
              </aside>
            </div>
          </div>
        </template>

        <template v-else>
          <div class="module-shell">
            <nav class="module-tabs blog-route-tabs" aria-label="Blogs sections">
              <button
                v-for="item in blogNav"
                :key="item.key"
                class="module-tab"
                :class="{ active: currentSection === item.key }"
                @click="goTo(item.path)"
              >
                {{ item.label }}
              </button>
            </nav>

            <div class="module-hero">
              <div>
                <p class="eyebrow">Publishing</p>
                <h1>{{ sectionTitle }}</h1>
                <p class="lead">{{ sectionLead }}</p>
              </div>
              <div class="module-actions">
                <button v-if="currentSection === 'posts'" class="topbar-button" @click="openNewPost">Add New Post</button>
              </div>
            </div>

            <section v-if="currentSection === 'posts'" class="surface-card workspace-table">
              <div class="blog-list-toolbar">
                <div class="blog-filter-strip">
                  <button v-for="item in postFilters" :key="item" class="module-tab" :class="{ active: postFilter === item }" @click="postFilter = item">{{ item }}</button>
                </div>
                <div class="blog-list-tools">
                  <select v-model="bulkAction" class="input blog-bulk-select">
                    <option value="">Bulk actions</option>
                    <option value="delete">Delete</option>
                    <option value="publish">Publish</option>
                    <option value="draft">Move to Draft</option>
                  </select>
                  <button class="topbar-button topbar-button-ghost" @click="runBulkAction" :disabled="!selectedPosts.length || !bulkAction">Apply</button>
                  <input v-model="searchTerm" class="input blog-search" placeholder="Search posts" />
                </div>
              </div>

              <table class="data-table">
                <thead>
                  <tr>
                    <th><input type="checkbox" :checked="allVisibleSelected" @change="toggleAllVisible" /></th>
                    <th>Title</th>
                    <th>Author</th>
                    <th>Categories</th>
                    <th>Tags</th>
                    <th>Status</th>
                    <th>Comments</th>
                    <th>Date</th>
                    <th>Actions</th>
                  </tr>
                </thead>
                <tbody>
                  <tr v-for="post in filteredPosts" :key="post.id">
                    <td><input type="checkbox" :value="post.id" v-model="selectedPosts" /></td>
                    <td><strong>{{ post.title }}</strong><div class="table-subtle">{{ post.slug }}</div></td>
                    <td>{{ post.author || 'Unassigned' }}</td>
                    <td>{{ categorySummary(post) }}</td>
                    <td>{{ tagSummary(post.tags) }}</td>
                    <td><span class="status-pill">{{ post.status }}</span></td>
                    <td>{{ commentCount(post.id) }}</td>
                    <td>{{ stamp(post.updatedAt || post.publishedAt) }}</td>
                    <td class="table-actions">
                      <button class="link-button" @click="editPost(post.id)">Edit</button>
                      <button class="link-button" @click="previewSite">View</button>
                      <button class="link-button" @click="duplicatePost(post.id)">Duplicate</button>
                      <button class="link-button" @click="togglePublish(post)">{{ post.status === 'published' ? 'Unpublish' : 'Publish' }}</button>
                      <button class="link-button danger-link" @click="trashPost(post.id)">Trash</button>
                    </td>
                  </tr>
                </tbody>
              </table>
              <div v-if="!filteredPosts.length" class="empty-module-state">
                <strong>No posts yet. Publish your first article.</strong>
                <button class="topbar-button" @click="openNewPost">Add New Post</button>
              </div>
            </section>

            <section v-else-if="currentSection === 'categories'" class="blog-settings-layout">
              <article class="surface-card compact-surface">
                <p class="eyebrow">Add category</p>
                <form class="modal-form" @submit.prevent="submitCategory">
                  <label class="field"><span>Name</span><input v-model="categoryForm.name" class="input" /></label>
                  <label class="field"><span>Slug</span><input v-model="categoryForm.slug" class="input" /></label>
                  <label class="field"><span>Description</span><textarea v-model="categoryForm.description" class="input textarea" rows="4"></textarea></label>
                  <button class="button" type="submit">Save Category</button>
                </form>
              </article>
              <article class="surface-card workspace-table">
                <table class="data-table">
                  <thead><tr><th>Name</th><th>Slug</th><th>Posts</th><th>Actions</th></tr></thead>
                  <tbody>
                    <tr v-for="category in blogs.categories" :key="category.id">
                      <td>{{ category.name }}</td>
                      <td>{{ category.slug }}</td>
                      <td>{{ postsUsingCategory(category.name) }}</td>
                      <td class="table-actions">
                        <button class="link-button" @click="Object.assign(categoryForm, category)">Edit</button>
                        <button class="link-button danger-link" @click="removeCategory(category.id)">Delete</button>
                      </td>
                    </tr>
                  </tbody>
                </table>
              </article>
            </section>

            <section v-else-if="currentSection === 'tags'" class="blog-settings-layout">
              <article class="surface-card compact-surface">
                <p class="eyebrow">Add tag</p>
                <form class="modal-form" @submit.prevent="submitTag">
                  <label class="field"><span>Name</span><input v-model="tagForm.name" class="input" /></label>
                  <label class="field"><span>Slug</span><input v-model="tagForm.slug" class="input" /></label>
                  <label class="field"><span>Visibility</span><select v-model="tagForm.visibility" class="input"><option>public</option><option>private</option></select></label>
                  <button class="button" type="submit">Save Tag</button>
                </form>
              </article>
              <article class="surface-card workspace-table">
                <table class="data-table">
                  <thead><tr><th>Name</th><th>Slug</th><th>Posts</th><th>Actions</th></tr></thead>
                  <tbody>
                    <tr v-for="tag in blogs.tags" :key="tag.id">
                      <td>{{ tag.name }}</td>
                      <td>{{ tag.slug }}</td>
                      <td>{{ postsUsingTag(tag.name) }}</td>
                      <td class="table-actions">
                        <button class="link-button" @click="Object.assign(tagForm, tag)">Edit</button>
                        <button class="link-button danger-link" @click="removeTag(tag.id)">Delete</button>
                      </td>
                    </tr>
                  </tbody>
                </table>
              </article>
            </section>

            <section v-else-if="currentSection === 'authors'" class="blog-settings-layout">
              <article class="surface-card compact-surface">
                <p class="eyebrow">Add author</p>
                <form class="modal-form" @submit.prevent="submitAuthor">
                  <label class="field"><span>Name</span><input v-model="authorForm.name" class="input" /></label>
                  <label class="field"><span>Email</span><input v-model="authorForm.email" class="input" /></label>
                  <label class="field"><span>Role</span><select v-model="authorForm.role" class="input"><option>author</option><option>editor</option><option>admin</option></select></label>
                  <label class="field"><span>Bio</span><textarea v-model="authorForm.bio" class="input textarea" rows="4"></textarea></label>
                  <button class="button" type="submit">Save Author</button>
                </form>
              </article>
              <article class="surface-card workspace-table">
                <table class="data-table">
                  <thead><tr><th>Name</th><th>Email</th><th>Role</th><th>Posts</th><th>Actions</th></tr></thead>
                  <tbody>
                    <tr v-for="author in blogs.authors" :key="author.id">
                      <td>{{ author.name }}</td>
                      <td>{{ author.email }}</td>
                      <td>{{ author.role }}</td>
                      <td>{{ postsUsingAuthor(author.name) }}</td>
                      <td class="table-actions">
                        <button class="link-button" @click="Object.assign(authorForm, author)">Edit</button>
                        <button class="link-button danger-link" @click="removeAuthor(author.id)">Delete</button>
                      </td>
                    </tr>
                  </tbody>
                </table>
              </article>
            </section>

            <section v-else-if="currentSection === 'media'" class="surface-card workspace-table">
              <div class="table-head"><strong>Media Library</strong><button class="topbar-button topbar-button-ghost" disabled>Upload</button></div>
              <div class="media-grid">
                <article v-for="item in mediaItems" :key="item.url" class="media-card">
                  <strong>{{ item.name }}</strong>
                  <span>{{ item.type }} · {{ item.size }}</span>
                  <button class="link-button" @click="copyText(item.url)">Copy URL</button>
                </article>
              </div>
            </section>

            <section v-else-if="currentSection === 'comments'" class="surface-card workspace-table">
              <table class="data-table">
                <thead><tr><th>Comment</th><th>Author</th><th>Post</th><th>Status</th><th>Date</th><th>Actions</th></tr></thead>
                <tbody>
                  <tr v-for="comment in blogs.comments" :key="comment.id">
                    <td>{{ comment.content }}</td>
                    <td>{{ comment.authorName }}</td>
                    <td>{{ comment.postTitle }}</td>
                    <td><span class="status-pill">{{ comment.status }}</span></td>
                    <td>{{ stamp(comment.createdAt) }}</td>
                    <td class="table-actions">
                      <button class="link-button" @click="moderateComment(comment.id, 'approve')">Approve</button>
                      <button class="link-button danger-link" @click="moderateComment(comment.id, 'trash')">Trash</button>
                    </td>
                  </tr>
                </tbody>
              </table>
            </section>

            <section v-else-if="currentSection === 'newsletter'" class="module-grid">
              <article class="surface-card compact-surface">
                <p class="eyebrow">Newsletter</p>
                <form class="modal-form" @submit.prevent="saveNewsletter">
                  <label class="setting-row"><span>Newsletter enabled</span><input v-model="newsletterForm.enabled" type="checkbox" /></label>
                  <label class="field"><span>Sender name</span><input v-model="newsletterForm.senderName" class="input" /></label>
                  <label class="field"><span>Sender email</span><input v-model="newsletterForm.senderEmail" class="input" /></label>
                  <label class="setting-row"><span>Send post as email</span><input v-model="newsletterForm.sendPostsAsEmail" type="checkbox" /></label>
                  <button class="button" type="submit">Save Newsletter</button>
                </form>
              </article>
              <article class="surface-card compact-surface">
                <p class="eyebrow">Campaign drafts</p>
                <h2>{{ blogs.members.length }} subscribers</h2>
                <p class="lead">Start collecting subscribers and send published posts as email campaigns later.</p>
              </article>
            </section>

            <section v-else-if="currentSection === 'members'" class="blog-settings-layout">
              <article class="surface-card compact-surface">
                <p class="eyebrow">Add member</p>
                <form class="modal-form" @submit.prevent="submitMember">
                  <label class="field"><span>Name</span><input v-model="memberForm.name" class="input" /></label>
                  <label class="field"><span>Email</span><input v-model="memberForm.email" class="input" /></label>
                  <label class="field"><span>Type</span><select v-model="memberForm.memberType" class="input"><option>free</option><option>paid</option></select></label>
                  <label class="field"><span>Status</span><select v-model="memberForm.status" class="input"><option>active</option><option>paused</option><option>invited</option></select></label>
                  <button class="button" type="submit">Save Member</button>
                </form>
              </article>
              <article class="surface-card workspace-table">
                <div class="table-head"><strong>Members</strong><button class="topbar-button topbar-button-ghost" disabled>Import</button></div>
                <table class="data-table">
                  <thead><tr><th>Member</th><th>Email</th><th>Type</th><th>Status</th><th>Joined</th><th>Actions</th></tr></thead>
                  <tbody>
                    <tr v-for="member in blogs.members" :key="member.id">
                      <td>{{ member.name }}</td>
                      <td>{{ member.email }}</td>
                      <td>{{ member.memberType }}</td>
                      <td><span class="status-pill">{{ member.status }}</span></td>
                      <td>{{ stamp(member.joinedAt) }}</td>
                      <td class="table-actions">
                        <button class="link-button" @click="Object.assign(memberForm, member)">Edit</button>
                        <button class="link-button danger-link" @click="removeMember(member.id)">Delete</button>
                      </td>
                    </tr>
                  </tbody>
                </table>
              </article>
            </section>

            <section v-else class="blog-settings-layout">
              <article class="surface-card compact-surface">
                <p class="eyebrow">Blog settings</p>
                <form class="modal-form" @submit.prevent="saveBlogSettings">
                  <label class="field"><span>Blog title</span><input v-model="settingsForm.blogTitle" class="input" /></label>
                  <label class="field"><span>Description</span><textarea v-model="settingsForm.description" class="input textarea" rows="4"></textarea></label>
                  <label class="field"><span>Default author</span><select v-model="settingsForm.defaultAuthorId" class="input"><option value="">Select author</option><option v-for="author in blogs.authors" :key="author.id" :value="author.id">{{ author.name }}</option></select></label>
                  <label class="field"><span>Canonical URL</span><input v-model="settingsForm.canonicalUrl" class="input" /></label>
                  <label class="field"><span>Posts per page</span><input v-model.number="settingsForm.postsPerPage" class="input" type="number" min="1" /></label>
                  <label class="setting-row"><span>RSS enabled</span><input v-model="settingsForm.rssEnabled" type="checkbox" /></label>
                  <label class="setting-row"><span>Comments enabled</span><input v-model="settingsForm.commentsEnabled" type="checkbox" /></label>
                  <label class="setting-row"><span>Social sharing enabled</span><input v-model="settingsForm.socialSharingEnabled" type="checkbox" /></label>
                  <button class="button" type="submit">Save Settings</button>
                </form>
              </article>
              <article class="surface-card compact-surface">
                <p class="eyebrow">Publishing</p>
                <h2>Ready for editorial work</h2>
                <p class="lead">Use this area to set the defaults that shape every future post and channel.</p>
              </article>
            </section>
          </div>
        </template>
      </section>
    </div>
  </main>
</template>

<script setup>
import { computed, onMounted, reactive, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useBlogsStore, blankPost, blankSettings } from '../stores/blogs'

const route = useRoute()
const router = useRouter()
const blogs = useBlogsStore()
const siteName = ref(localStorage.getItem('sitecraft_active_site_name') || 'SiteCraft')
const searchTerm = ref('')
const postFilter = ref('All')
const bulkAction = ref('')
const selectedPosts = ref([])
const editorCategories = ref('')

const categoryForm = reactive({ id: '', name: '', slug: '', description: '' })
const tagForm = reactive({ id: '', name: '', slug: '', visibility: 'public' })
const authorForm = reactive({ id: '', name: '', email: '', role: 'author', bio: '' })
const memberForm = reactive({ id: '', name: '', email: '', memberType: 'free', status: 'active' })
const newsletterForm = reactive({ enabled: true, senderName: '', senderEmail: '', sendPostsAsEmail: true })
const settingsForm = reactive(blankSettings())
const editorForm = reactive(blankPost())

const blogNav = [
  { key: 'posts', label: 'All Posts', path: '/blogs/posts' },
  { key: 'new', label: 'Add New', path: '/blogs/posts/new' },
  { key: 'categories', label: 'Categories', path: '/blogs/categories' },
  { key: 'tags', label: 'Tags', path: '/blogs/tags' },
  { key: 'authors', label: 'Authors', path: '/blogs/authors' },
  { key: 'media', label: 'Media', path: '/blogs/media' },
  { key: 'comments', label: 'Comments', path: '/blogs/comments' },
  { key: 'newsletter', label: 'Newsletter', path: '/blogs/newsletter' },
  { key: 'members', label: 'Members', path: '/blogs/members' },
  { key: 'settings', label: 'Settings', path: '/blogs/settings' }
]
const siteSections = [
  { key: 'overview', label: 'Overview' },
  { key: 'analytics', label: 'Analytics' },
  { key: 'pages', label: 'Pages' },
  { key: 'ecommerce', label: 'Ecommerce' },
  { key: 'subscriptions', label: 'Subscription Stores' },
  { key: 'blogs', label: 'Blogs' },
  { key: 'deployments', label: 'Deployments' },
  { key: 'settings', label: 'Settings' }
]
const postFilters = ['All', 'Published', 'Drafts', 'Scheduled', 'Trash']

const resolvedSiteId = computed(() => String(route.query.siteId || localStorage.getItem('sitecraft_active_site_id') || ''))
const isEditorRoute = computed(() => route.path === '/blogs/posts/new' || /^\/blogs\/posts\/.+\/edit$/.test(route.path))
const currentSection = computed(() => {
  if (route.path === '/blogs/posts/new') return 'new'
  if (route.path.startsWith('/blogs/posts/') || route.path === '/blogs/posts' || route.path === '/blogs') return 'posts'
  if (route.path.startsWith('/blogs/categories')) return 'categories'
  if (route.path.startsWith('/blogs/tags')) return 'tags'
  if (route.path.startsWith('/blogs/authors')) return 'authors'
  if (route.path.startsWith('/blogs/media')) return 'media'
  if (route.path.startsWith('/blogs/comments')) return 'comments'
  if (route.path.startsWith('/blogs/newsletter')) return 'newsletter'
  if (route.path.startsWith('/blogs/members')) return 'members'
  return 'settings'
})
const sectionTitle = computed(() => ({
  posts: 'Posts',
  categories: 'Categories',
  tags: 'Tags',
  authors: 'Authors',
  media: 'Media Library',
  comments: 'Comments',
  newsletter: 'Newsletter',
  members: 'Members',
  settings: 'Blog Settings'
}[currentSection.value]))
const sectionLead = computed(() => ({
  posts: 'Manage the entire editorial queue with a compact CMS workflow.',
  categories: 'Organize posts into clear content groupings.',
  tags: 'Maintain lightweight topical tags for discovery.',
  authors: 'Manage editorial ownership and bylines.',
  media: 'Review featured assets and copyable URLs.',
  comments: 'Moderate the latest audience responses.',
  newsletter: 'Prepare subscriber communications and sender defaults.',
  members: 'Manage free and paid members tied to the blog.',
  settings: 'Control the defaults that shape the publication.'
}[currentSection.value]))

const slugPreview = computed(() => editorForm.title.trim().toLowerCase().replace(/[^a-z0-9]+/g, '-').replace(/(^-|-$)/g, '') || 'new-post')
const filteredPosts = computed(() => blogs.posts.filter((post) => {
  const search = searchTerm.value.trim().toLowerCase()
  const matchesSearch = !search || `${post.title} ${post.slug} ${post.author || ''}`.toLowerCase().includes(search)
  const normalized = String(post.status || '').toLowerCase()
  const matchesFilter = postFilter.value === 'All'
    || (postFilter.value === 'Published' && normalized === 'published')
    || (postFilter.value === 'Drafts' && normalized === 'draft')
    || (postFilter.value === 'Scheduled' && normalized === 'scheduled')
    || (postFilter.value === 'Trash' && normalized === 'trash')
  return matchesSearch && matchesFilter
}))
const allVisibleSelected = computed(() => filteredPosts.value.length > 0 && filteredPosts.value.every((post) => selectedPosts.value.includes(post.id)))
const mediaItems = computed(() => blogs.posts.filter((post) => post.featuredImage).map((post) => ({
  name: post.title,
  type: 'image',
  size: 'remote',
  url: post.featuredImage
})))

const stamp = (value) => value ? new Date(value).toLocaleDateString() : 'Just now'
const tagSummary = (tags) => Array.isArray(tags) ? tags.join(', ') : String(tags || '').replace(/[[\]"]/g, '')
const categorySummary = () => blogs.categories.slice(0, 2).map((item) => item.name).join(', ') || 'Unassigned'
const commentCount = (postId) => blogs.comments.filter((item) => item.postId === postId).length || blogs.comments.filter((item) => item.postTitle).length
const postsUsingCategory = (name) => blogs.posts.filter((post) => String(post.content || '').includes(name) || String(post.excerpt || '').includes(name)).length
const postsUsingTag = (name) => blogs.posts.filter((post) => tagSummary(post.tags).toLowerCase().includes(name.toLowerCase())).length
const postsUsingAuthor = (name) => blogs.posts.filter((post) => post.author === name).length

const syncSettings = () => {
  Object.assign(newsletterForm, {
    enabled: blogs.newsletterSettings?.enabled ?? true,
    senderName: blogs.newsletterSettings?.senderName || '',
    senderEmail: blogs.newsletterSettings?.senderEmail || '',
    sendPostsAsEmail: blogs.newsletterSettings?.sendPostsAsEmail ?? true
  })
  Object.assign(settingsForm, blankSettings(), blogs.blogSettings || {})
}

const syncEditor = async () => {
  if (route.path === '/blogs/posts/new') {
    Object.assign(editorForm, blankPost())
    editorCategories.value = ''
    return
  }
  const match = route.path.match(/^\/blogs\/posts\/(.+)\/edit$/)
  if (!match || !resolvedSiteId.value) return
  const post = await blogs.loadPost(resolvedSiteId.value, match[1])
  if (post) {
    Object.assign(editorForm, blankPost(), post)
    editorCategories.value = blogs.categories.slice(0, 2).map((item) => item.name).join(', ')
  }
}

const goTo = (path) => router.push({ path, query: { siteId: resolvedSiteId.value } })
const backToSite = () => router.push(resolvedSiteId.value ? `/site/${resolvedSiteId.value}` : '/dashboard')
const handleSiteSectionClick = async (key) => {
  if (key === 'blogs') return
  if (!resolvedSiteId.value) {
    await router.push('/dashboard')
    return
  }
  await router.push(`/site/${resolvedSiteId.value}`)
}
const openNewPost = () => goTo('/blogs/posts/new')
const editPost = (id) => goTo(`/blogs/posts/${id}/edit`)
const previewSite = () => {
  const url = localStorage.getItem('sitecraft_active_site_url')
  if (url) window.open(url, '_blank', 'noopener')
}
const saveDraft = async () => {
  await blogs.save(resolvedSiteId.value, { ...editorForm, slug: editorForm.slug || slugPreview.value, status: 'draft' })
  if (!editorForm.id && blogs.posts[0]?.id) {
    Object.assign(editorForm, blankPost(), blogs.posts[0])
    await goTo(`/blogs/posts/${blogs.posts[0].id}/edit`)
  }
}
const publishPost = async () => {
  await blogs.save(resolvedSiteId.value, { ...editorForm, slug: editorForm.slug || slugPreview.value, status: editorForm.status === 'scheduled' ? 'scheduled' : 'published' })
  const activeId = editorForm.id || blogs.posts[0]?.id
  if (activeId && editorForm.status !== 'scheduled') {
    await blogs.setPublished(resolvedSiteId.value, activeId, true)
    const post = blogs.posts.find((item) => item.id === activeId)
    if (post) Object.assign(editorForm, blankPost(), post)
  }
}
const duplicatePost = async (id) => { await blogs.duplicate(resolvedSiteId.value, id) }
const trashPost = async (id) => { await blogs.trash(resolvedSiteId.value, id) }
const togglePublish = async (post) => { await blogs.setPublished(resolvedSiteId.value, post.id, post.status !== 'published') }
const toggleAllVisible = () => {
  selectedPosts.value = allVisibleSelected.value ? [] : filteredPosts.value.map((post) => post.id)
}
const runBulkAction = async () => {
  for (const id of selectedPosts.value) {
    if (bulkAction.value === 'delete') await blogs.trash(resolvedSiteId.value, id)
    if (bulkAction.value === 'publish') await blogs.setPublished(resolvedSiteId.value, id, true)
    if (bulkAction.value === 'draft') await blogs.setPublished(resolvedSiteId.value, id, false)
  }
  selectedPosts.value = []
  bulkAction.value = ''
}
const submitCategory = async () => {
  await blogs.saveCategory(resolvedSiteId.value, { ...categoryForm })
  Object.assign(categoryForm, { id: '', name: '', slug: '', description: '' })
}
const removeCategory = async (id) => { await blogs.removeCategory(resolvedSiteId.value, id) }
const submitTag = async () => {
  await blogs.saveTag(resolvedSiteId.value, { ...tagForm })
  Object.assign(tagForm, { id: '', name: '', slug: '', visibility: 'public' })
}
const removeTag = async (id) => { await blogs.removeTag(resolvedSiteId.value, id) }
const submitAuthor = async () => {
  await blogs.saveAuthor(resolvedSiteId.value, { ...authorForm })
  Object.assign(authorForm, { id: '', name: '', email: '', role: 'author', bio: '' })
}
const removeAuthor = async (id) => { await blogs.removeAuthor(resolvedSiteId.value, id) }
const saveNewsletter = async () => { await blogs.saveNewsletterSettings(resolvedSiteId.value, { ...newsletterForm }) }
const submitMember = async () => {
  await blogs.saveMember(resolvedSiteId.value, { ...memberForm })
  Object.assign(memberForm, { id: '', name: '', email: '', memberType: 'free', status: 'active' })
}
const removeMember = async (id) => { await blogs.removeMember(resolvedSiteId.value, id) }
const moderateComment = async (id, action) => { await blogs.moderateComment(resolvedSiteId.value, id, action) }
const saveBlogSettings = async () => { await blogs.saveBlogSettings(resolvedSiteId.value, { ...settingsForm }) }
const copyText = async (value) => { if (value) await navigator.clipboard.writeText(value) }

watch(() => resolvedSiteId.value, async (siteId) => {
  if (!siteId) return
  localStorage.setItem('sitecraft_active_site_id', siteId)
  await blogs.load(siteId)
  syncSettings()
  await syncEditor()
}, { immediate: true })

watch(() => route.path, async () => {
  blogs.clearFeedback()
  await syncEditor()
})

onMounted(async () => {
  if (resolvedSiteId.value) {
    await blogs.load(resolvedSiteId.value)
    syncSettings()
    await syncEditor()
  }
})
</script>
