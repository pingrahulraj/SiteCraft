<template>
  <main class="app-shell">
    <header class="topbar">
      <div class="topbar-brand">
        <span class="brand-mark brand-mark-sm"></span>
        <div>
          <strong>{{ site?.name || 'Site' }}</strong>
          <span>Site dashboard</span>
        </div>
      </div>
      <div class="topbar-actions">
        <button class="topbar-button" @click="editSite">Edit site</button>
        <button class="topbar-button topbar-button-ghost" @click="backToWorkspace">Workspace</button>
        <button class="topbar-button topbar-button-ghost" @click="signOut">Sign out</button>
      </div>
    </header>

    <div class="workspace">
      <aside class="sidebar sidebar-site">
        <div class="sidebar-section">
          <p class="sidebar-label">Site navigation</p>
          <nav class="sidebar-nav">
            <button
              v-for="item in sections"
              :key="item.key"
              class="sidebar-link"
              :class="{ active: activeSection === item.key }"
              @click="activeSection = item.key"
            >
              {{ item.label }}
            </button>
          </nav>
        </div>

        <div class="sidebar-section sidebar-card">
          <p class="sidebar-label">Project</p>
          <strong>{{ site?.name || 'Loading site...' }}</strong>
          <p class="sidebar-copy">{{ site?.description || 'Loading project context.' }}</p>
        </div>
      </aside>

      <section class="content">
        <template v-if="activeSection === 'ecommerce'">
          <div class="module-shell">
            <div class="module-hero">
              <div>
                <p class="eyebrow">Ecommerce</p>
                <h1>Ecommerce</h1>
                <p class="lead">Manage products, orders, discounts, shipping, and store settings.</p>
              </div>
              <div class="module-actions">
                <button class="topbar-button" @click="openProductModal()">+ Add Product</button>
                <button class="topbar-button topbar-button-ghost">Import</button>
                <button class="topbar-button topbar-button-ghost" @click="ecommerceTab = 'Store Settings'">Store Settings</button>
              </div>
            </div>

            <div class="module-kpis">
              <div class="module-kpi"><span>Products</span><strong>{{ ecommerce.products.length }}</strong><small>Catalog items</small></div>
              <div class="module-kpi"><span>Orders</span><strong>12</strong><small>Last 30 days</small></div>
              <div class="module-kpi"><span>Revenue</span><strong>$4,860</strong><small>Mock gross sales</small></div>
              <div class="module-kpi"><span>Conversion</span><strong>3.4%</strong><small>Store conversion</small></div>
            </div>

            <div class="module-tabs">
              <button v-for="tab in ecommerceTabs" :key="tab" class="module-tab" :class="{ active: ecommerceTab === tab }" @click="ecommerceTab = tab">{{ tab }}</button>
            </div>

            <section v-if="ecommerceTab === 'Products'" class="surface-card workspace-table">
              <div class="table-head">
                <div>
                  <strong>Products</strong>
                  <div class="table-meta">Create, price, and organize your catalog.</div>
                </div>
                <button class="topbar-button" @click="openProductModal()">Add Product</button>
              </div>
              <p v-if="ecommerce.error" class="error">{{ ecommerce.error }}</p>
              <table v-if="ecommerce.products.length" class="data-table">
                <thead>
                  <tr>
                    <th>Product</th>
                    <th>SKU</th>
                    <th>Price</th>
                    <th>Stock</th>
                    <th>Status</th>
                    <th>Updated</th>
                    <th>Actions</th>
                  </tr>
                </thead>
                <tbody>
                  <tr v-for="product in ecommerceProducts" :key="product.id">
                    <td>
                      <strong>{{ product.name }}</strong>
                      <div class="table-subtle">{{ product.slug }}</div>
                    </td>
                    <td>{{ product.sku }}</td>
                    <td>{{ currency(product.price) }}</td>
                    <td>{{ product.stock }}</td>
                    <td><span class="status-pill">{{ product.status }}</span></td>
                    <td>{{ stamp(product.updatedAt) }}</td>
                    <td class="table-actions">
                      <button class="link-button" @click="openProductModal(product)">Edit</button>
                      <button class="link-button" @click="duplicateProduct(product.id)">Duplicate</button>
                      <button class="link-button danger-link" @click="removeProduct(product.id)">Delete</button>
                    </td>
                  </tr>
                </tbody>
              </table>
              <div v-else class="empty-module-state">
                <strong>No products yet. Add your first product to start selling.</strong>
                <button class="topbar-button" @click="openProductModal()">Add Product</button>
              </div>
            </section>

            <section v-else-if="ecommerceTab === 'Orders'" class="surface-card workspace-table">
              <div class="table-head"><strong>Orders</strong><span class="table-meta">Track payment and fulfillment status.</span></div>
              <table v-if="orderRows.length" class="data-table">
                <thead>
                  <tr><th>Order</th><th>Customer</th><th>Amount</th><th>Payment</th><th>Fulfillment</th><th>Status</th><th>Date</th></tr>
                </thead>
                <tbody>
                  <tr v-for="order in orderRows" :key="order.id">
                    <td>{{ order.orderNumber }}</td>
                    <td>{{ order.customerName }}</td>
                    <td>{{ currency(order.amount) }}</td>
                    <td>{{ order.paymentStatus }}</td>
                    <td>{{ order.fulfillmentStatus }}</td>
                    <td><span class="status-pill">{{ order.status }}</span></td>
                    <td>{{ stamp(order.updatedAt) }}</td>
                  </tr>
                </tbody>
              </table>
              <div v-else class="empty-module-state"><strong>Orders will appear here once customers start buying.</strong></div>
            </section>

            <section v-else-if="ecommerceTab === 'Customers'" class="surface-card workspace-table">
              <div class="table-head"><strong>Customers</strong><button class="topbar-button topbar-button-ghost" disabled>Export</button></div>
              <table v-if="customerRows.length" class="data-table">
                <thead>
                  <tr><th>Customer</th><th>Email</th><th>Orders</th><th>Total Spent</th><th>Last Order</th></tr>
                </thead>
                <tbody>
                  <tr v-for="customer in customerRows" :key="customer.email">
                    <td>{{ customer.name }}</td>
                    <td>{{ customer.email }}</td>
                    <td>{{ customer.ordersCount }}</td>
                    <td>{{ currency(customer.totalSpent) }}</td>
                    <td>{{ stamp(customer.lastOrderAt) }}</td>
                  </tr>
                </tbody>
              </table>
              <div v-else class="empty-module-state"><strong>No customers yet.</strong><span>Customer profiles will build automatically as orders arrive.</span></div>
            </section>

            <section v-else-if="ecommerceTab === 'Coupons'" class="surface-card workspace-table">
              <div class="table-head"><strong>Coupons</strong><button class="topbar-button topbar-button-ghost">Add Coupon</button></div>
              <table class="data-table">
                <thead>
                  <tr><th>Code</th><th>Type</th><th>Value</th><th>Expiry</th><th>Usage</th><th>Status</th></tr>
                </thead>
                <tbody>
                  <tr v-for="coupon in couponRows" :key="coupon.code">
                    <td>{{ coupon.code }}</td>
                    <td>{{ coupon.discountType }}</td>
                    <td>{{ coupon.value }}</td>
                    <td>{{ stamp(coupon.expiresAt) }}</td>
                    <td>{{ coupon.usageCount }}</td>
                    <td><span class="status-pill">{{ coupon.status }}</span></td>
                  </tr>
                </tbody>
              </table>
            </section>

            <section v-else-if="ecommerceTab === 'Shipping'" class="module-grid">
              <article class="surface-card compact-surface" v-for="zone in shippingZoneCards" :key="zone.title">
                <p class="eyebrow">{{ zone.eyebrow }}</p>
                <h2>{{ zone.title }}</h2>
                <p class="lead">{{ zone.lead }}</p>
              </article>
            </section>

            <section v-else-if="ecommerceTab === 'Taxes'" class="module-grid">
              <article class="surface-card compact-surface">
                <p class="eyebrow">Tax engine</p>
                <h2>Tax settings</h2>
                <div class="settings-stack">
                  <label class="setting-row"><span>Enable taxes</span><strong>{{ ecommerce.taxSettings?.taxesEnabled ? 'Enabled' : 'Disabled' }}</strong></label>
                  <label class="setting-row"><span>Prices include tax</span><strong>{{ ecommerce.taxSettings?.pricesIncludeTax ? 'Yes' : 'No' }}</strong></label>
                  <label class="setting-row"><span>Default tax region</span><strong>{{ ecommerce.taxSettings?.defaultRegion || 'US' }}</strong></label>
                </div>
              </article>
              <article class="surface-card compact-surface">
                <p class="eyebrow">Tax rates</p>
                <h2>Region placeholders</h2>
                <table class="data-table compact-table">
                  <thead><tr><th>Region</th><th>Rate</th><th>Mode</th></tr></thead>
                  <tbody>
                    <tr><td>US</td><td>8.25%</td><td>Standard</td></tr>
                    <tr><td>EU</td><td>20%</td><td>VAT</td></tr>
                  </tbody>
                </table>
              </article>
            </section>

            <section v-else class="module-grid">
              <article class="surface-card compact-surface" v-for="setting in storeSettingCards" :key="setting.title">
                <p class="eyebrow">{{ setting.eyebrow }}</p>
                <h2>{{ setting.title }}</h2>
                <p class="lead">{{ setting.lead }}</p>
              </article>
            </section>
          </div>
        </template>

        <template v-else-if="activeSection === 'subscriptions'">
          <div class="module-shell">
            <div class="module-hero">
              <div>
                <p class="eyebrow">Subscription Stores</p>
                <h1>Subscription Stores</h1>
                <p class="lead">Sell recurring products, memberships, or subscription boxes.</p>
              </div>
              <div class="module-actions">
                <button class="topbar-button" @click="openPlanModal()">+ Create Plan</button>
                <button class="topbar-button topbar-button-ghost" @click="subscriptionTab = 'Settings'">Settings</button>
              </div>
            </div>

            <div class="module-kpis">
              <div class="module-kpi"><span>Active subscriptions</span><strong>{{ subscriptionRows.length }}</strong><small>Across all plans</small></div>
              <div class="module-kpi"><span>MRR</span><strong>{{ currency(subscriptionRows.reduce((sum, item) => sum + Number(item.amount || 0), 0)) }}</strong><small>Recurring revenue</small></div>
              <div class="module-kpi"><span>Failed payments</span><strong>{{ failedPaymentRows.length }}</strong><small>Recovery needed</small></div>
              <div class="module-kpi"><span>Churn</span><strong>2.1%</strong><small>Trailing 30 days</small></div>
            </div>

            <div class="module-tabs">
              <button v-for="tab in subscriptionTabs" :key="tab" class="module-tab" :class="{ active: subscriptionTab === tab }" @click="subscriptionTab = tab">{{ tab }}</button>
            </div>

            <section v-if="subscriptionTab === 'Plans'" class="surface-card workspace-table">
              <div class="table-head"><strong>Plans</strong><button class="topbar-button" @click="openPlanModal()">Create Plan</button></div>
              <table v-if="subscriptionsStore.plans.length" class="data-table">
                <thead><tr><th>Plan</th><th>Price</th><th>Interval</th><th>Trial</th><th>Status</th><th>Actions</th></tr></thead>
                <tbody>
                  <tr v-for="plan in subscriptionsStore.plans" :key="plan.id">
                    <td><strong>{{ plan.name }}</strong><div class="table-subtle">{{ plan.description }}</div></td>
                    <td>{{ currency(plan.price) }}</td>
                    <td>{{ plan.interval }}</td>
                    <td>{{ plan.trial }} days</td>
                    <td><span class="status-pill">{{ plan.status }}</span></td>
                    <td class="table-actions">
                      <button class="link-button" @click="openPlanModal(plan)">Edit</button>
                      <button class="link-button danger-link" @click="removePlan(plan.id)">Delete</button>
                    </td>
                  </tr>
                </tbody>
              </table>
              <div v-else class="empty-module-state">
                <strong>Create your first subscription plan.</strong>
                <button class="topbar-button" @click="openPlanModal()">Create Plan</button>
              </div>
            </section>

            <section v-else-if="subscriptionTab === 'Subscribers'" class="surface-card workspace-table">
              <div class="table-head"><strong>Subscribers</strong></div>
              <table class="data-table">
                <thead><tr><th>Subscriber</th><th>Email</th><th>Plan</th><th>Status</th><th>Joined</th></tr></thead>
                <tbody>
                  <tr v-for="subscriber in subscriberRows" :key="subscriber.email">
                    <td>{{ subscriber.name }}</td>
                    <td>{{ subscriber.email }}</td>
                    <td>{{ subscriber.plan || 'Plan linked later' }}</td>
                    <td><span class="status-pill">{{ subscriber.status }}</span></td>
                    <td>{{ stamp(subscriber.joinedAt) }}</td>
                  </tr>
                </tbody>
              </table>
            </section>

            <section v-else-if="subscriptionTab === 'Subscriptions'" class="surface-card workspace-table">
              <div class="table-head"><strong>Subscriptions</strong></div>
              <table class="data-table">
                <thead><tr><th>Customer</th><th>Plan</th><th>Renewal Date</th><th>Status</th><th>Amount</th></tr></thead>
                <tbody>
                  <tr v-for="subscription in subscriptionRows" :key="subscription.id">
                    <td>{{ subscription.subscriber }}</td>
                    <td>{{ subscription.plan }}</td>
                    <td>{{ stamp(subscription.renewalDate) }}</td>
                    <td><span class="status-pill">{{ subscription.status }}</span></td>
                    <td>{{ currency(subscription.amount) }}</td>
                  </tr>
                </tbody>
              </table>
            </section>

            <section v-else-if="subscriptionTab === 'Invoices'" class="surface-card workspace-table">
              <div class="table-head"><strong>Invoices</strong></div>
              <table class="data-table">
                <thead><tr><th>Invoice</th><th>Customer</th><th>Amount</th><th>Status</th><th>Date</th></tr></thead>
                <tbody>
                  <tr v-for="invoice in invoiceRows" :key="invoice.id">
                    <td>{{ invoice.invoiceNumber }}</td>
                    <td>{{ invoice.subscriber }}</td>
                    <td>{{ currency(invoice.amount) }}</td>
                    <td><span class="status-pill">{{ invoice.status }}</span></td>
                    <td>{{ stamp(invoice.issuedAt) }}</td>
                  </tr>
                </tbody>
              </table>
            </section>

            <section v-else-if="subscriptionTab === 'Failed Payments'" class="module-grid">
              <article class="surface-card compact-surface">
                <p class="eyebrow">Recovery</p>
                <h2>Retry rules</h2>
                <p class="lead">Failed payment recovery helps reduce lost revenue.</p>
              </article>
              <article class="surface-card compact-surface">
                <p class="eyebrow">Queue</p>
                <h2>Failed payment list</h2>
                <p class="lead">{{ failedPaymentRows.length }} subscriptions need retry emails and payment method recovery.</p>
              </article>
            </section>

            <section v-else class="module-grid">
              <article class="surface-card compact-surface" v-for="setting in subscriptionSettingCards" :key="setting.title">
                <p class="eyebrow">{{ setting.eyebrow }}</p>
                <h2>{{ setting.title }}</h2>
                <p class="lead">{{ setting.lead }}</p>
              </article>
            </section>
          </div>
        </template>

        <template v-else-if="activeSection === 'blogs'">
          <div class="module-shell">
            <div class="module-hero">
              <div>
                <p class="eyebrow">Blogs</p>
                <h1>Blogs</h1>
                <p class="lead">Publish articles, grow an audience, and prepare newsletter/member features.</p>
              </div>
              <div class="module-actions">
                <button class="topbar-button" @click="openPostModal()">+ New Post</button>
                <button class="topbar-button topbar-button-ghost" @click="blogsTab = 'Categories'">Categories</button>
                <button class="topbar-button topbar-button-ghost" @click="blogsTab = 'Settings'">Settings</button>
              </div>
            </div>

            <div class="module-kpis">
              <div class="module-kpi"><span>Posts</span><strong>{{ blogs.posts.length }}</strong><small>Published and draft</small></div>
              <div class="module-kpi"><span>Views</span><strong>{{ blogs.analytics?.views || 0 }}</strong><small>Total views</small></div>
              <div class="module-kpi"><span>Subscribers</span><strong>{{ blogs.members.length }}</strong><small>Newsletter growth</small></div>
              <div class="module-kpi"><span>Top Post</span><strong>{{ blogs.posts[0]?.title || 'No posts yet' }}</strong><small>Highest performing</small></div>
            </div>

            <div class="module-tabs">
              <button v-for="tab in blogTabs" :key="tab" class="module-tab" :class="{ active: blogsTab === tab }" @click="blogsTab = tab">{{ tab }}</button>
            </div>

            <section v-if="blogsTab === 'Posts'" class="surface-card workspace-table">
              <div class="table-head"><strong>Posts</strong><button class="topbar-button" @click="openPostModal()">New Post</button></div>
              <table v-if="blogs.posts.length" class="data-table">
                <thead>
                  <tr><th>Title</th><th>Slug</th><th>Author</th><th>Status</th><th>Updated</th><th>Actions</th></tr>
                </thead>
                <tbody>
                  <tr v-for="post in blogPosts" :key="post.id">
                    <td><strong>{{ post.title }}</strong><div class="table-subtle">{{ post.excerpt }}</div></td>
                    <td>{{ post.slug }}</td>
                    <td>{{ post.author }}</td>
                    <td><span class="status-pill">{{ post.status }}</span></td>
                    <td>{{ stamp(post.updatedAt) }}</td>
                    <td class="table-actions">
                      <button class="link-button" @click="openPostModal(post)">Edit</button>
                      <button class="link-button" @click="togglePost(post)">{{ post.status === 'published' ? 'Unpublish' : 'Publish' }}</button>
                      <button class="link-button danger-link" @click="removePost(post.id)">Delete</button>
                    </td>
                  </tr>
                </tbody>
              </table>
              <div v-else class="empty-module-state">
                <strong>No posts yet. Publish your first article.</strong>
                <button class="topbar-button" @click="openPostModal()">New Post</button>
              </div>
            </section>

            <section v-else-if="blogsTab === 'Categories'" class="surface-card workspace-table">
              <div class="table-head"><strong>Categories</strong><button class="topbar-button topbar-button-ghost">Add Category</button></div>
              <table class="data-table">
                <thead><tr><th>Name</th><th>Slug</th><th>Posts</th><th>Actions</th></tr></thead>
                <tbody>
                  <tr v-for="category in blogs.categories" :key="category.slug">
                    <td>{{ category.name }}</td>
                    <td>{{ category.slug }}</td>
                    <td>-</td>
                    <td><button class="link-button">Edit</button></td>
                  </tr>
                </tbody>
              </table>
            </section>

            <section v-else-if="blogsTab === 'Tags'" class="surface-card workspace-table">
              <div class="table-head"><strong>Tags</strong><button class="topbar-button topbar-button-ghost">Add Tag</button></div>
              <table class="data-table">
                <thead><tr><th>Tag</th><th>Slug</th><th>Posts</th><th>Visibility</th><th>Actions</th></tr></thead>
                <tbody>
                  <tr v-for="tag in blogs.tags" :key="tag.slug">
                    <td>{{ tag.name }}</td>
                    <td>{{ tag.slug }}</td>
                    <td>-</td>
                    <td>{{ tag.visibility }}</td>
                    <td><button class="link-button">Edit</button></td>
                  </tr>
                </tbody>
              </table>
            </section>

            <section v-else-if="blogsTab === 'Authors'" class="surface-card workspace-table">
              <div class="table-head"><strong>Authors</strong><button class="topbar-button topbar-button-ghost">Add Author</button></div>
              <table class="data-table">
                <thead><tr><th>Name</th><th>Email</th><th>Role</th><th>Posts</th></tr></thead>
                <tbody>
                  <tr v-for="author in blogs.authors" :key="author.email">
                    <td>{{ author.name }}</td>
                    <td>{{ author.email }}</td>
                    <td>{{ author.role }}</td>
                    <td>-</td>
                  </tr>
                </tbody>
              </table>
            </section>

            <section v-else-if="blogsTab === 'Newsletter'" class="module-grid">
              <article class="surface-card compact-surface">
                <p class="eyebrow">Newsletter</p>
                <h2>Send updates by email</h2>
                <div class="settings-stack">
                  <label class="setting-row"><span>Newsletter enabled</span><strong>{{ blogs.newsletterSettings?.enabled ? 'Enabled' : 'Disabled' }}</strong></label>
                  <label class="setting-row"><span>Subscriber count</span><strong>{{ blogs.members.length }}</strong></label>
                  <label class="setting-row"><span>Default sender</span><strong>{{ blogs.newsletterSettings?.senderEmail || 'hello@sitecraft.app' }}</strong></label>
                  <label class="setting-row"><span>Send post as email</span><strong>{{ blogs.newsletterSettings?.sendPostsAsEmail ? 'On' : 'Off' }}</strong></label>
                </div>
              </article>
              <article class="surface-card compact-surface">
                <p class="eyebrow">Next step</p>
                <h2>Grow your audience</h2>
                <p class="lead">Start collecting subscribers.</p>
                <button class="topbar-button">Create signup form</button>
              </article>
            </section>

            <section v-else-if="blogsTab === 'Members'" class="surface-card workspace-table">
              <div class="table-head"><strong>Members</strong><button class="topbar-button topbar-button-ghost">Import subscribers</button></div>
              <table class="data-table">
                <thead><tr><th>Member</th><th>Email</th><th>Type</th><th>Status</th><th>Joined</th></tr></thead>
                <tbody>
                  <tr v-for="member in blogs.members" :key="member.email">
                    <td>{{ member.name }}</td>
                    <td>{{ member.email }}</td>
                    <td>{{ member.memberType }}</td>
                    <td><span class="status-pill">{{ member.status }}</span></td>
                    <td>{{ stamp(member.joinedAt) }}</td>
                  </tr>
                </tbody>
              </table>
            </section>

            <section v-else-if="blogsTab === 'Analytics'" class="module-grid">
              <article class="surface-card compact-surface" v-for="card in blogAnalyticsCards" :key="card.title">
                <p class="eyebrow">{{ card.eyebrow }}</p>
                <h2>{{ card.title }}</h2>
                <p class="lead">{{ card.lead }}</p>
              </article>
            </section>

            <section v-else class="module-grid">
              <article class="surface-card compact-surface" v-for="card in blogSettingCards" :key="card.title">
                <p class="eyebrow">{{ card.eyebrow }}</p>
                <h2>{{ card.title }}</h2>
                <p class="lead">{{ card.lead }}</p>
              </article>
            </section>
          </div>
        </template>

        <template v-else>
          <div class="content-hero">
            <div>
              <p class="eyebrow">{{ sectionMeta.eyebrow }}</p>
              <h1>{{ sectionMeta.title }}</h1>
              <p class="lead">{{ sectionMeta.lead }}</p>
            </div>
            <div class="hero-metrics">
              <div class="metric"><span>Views</span><strong>{{ sectionMeta.metrics[0] }}</strong></div>
              <div class="metric"><span>Signal</span><strong>{{ sectionMeta.metrics[1] }}</strong></div>
              <div class="metric"><span>Status</span><strong>{{ site?.status || 'draft' }}</strong></div>
            </div>
          </div>

          <section class="dashboard-grid">
            <article v-for="card in sectionCards" :key="card.title" class="surface-card" :class="{ accent: card.accent }">
              <p class="eyebrow">{{ card.eyebrow }}</p>
              <h2>{{ card.title }}</h2>
              <p class="lead">{{ card.lead }}</p>
            </article>
          </section>
        </template>
      </section>
    </div>

    <div v-if="showProductModal" class="modal-backdrop" @click.self="closeProductModal">
      <div class="modal-shell modal-shell-wide">
        <div class="modal-header">
          <div>
            <p class="eyebrow">Ecommerce</p>
            <h2>{{ productForm.id ? 'Edit product' : 'Add product' }}</h2>
          </div>
          <button class="icon-button" @click="closeProductModal">x</button>
        </div>
        <form class="modal-form modal-form-grid" @submit.prevent="saveProduct">
          <label class="field"><span>Name</span><input v-model="productForm.name" class="input" required /></label>
          <label class="field"><span>Slug</span><input v-model="productForm.slug" class="input" required /></label>
          <label class="field modal-span-2"><span>Description</span><textarea v-model="productForm.description" class="input textarea" rows="4"></textarea></label>
          <label class="field"><span>Price</span><input v-model.number="productForm.price" class="input" type="number" min="0" step="0.01" /></label>
          <label class="field"><span>SKU</span><input v-model="productForm.sku" class="input" /></label>
          <label class="field"><span>Image URL</span><input v-model="productForm.imageUrl" class="input" placeholder="https://example.com/image.jpg" /></label>
          <label class="field"><span>Category</span><input v-model="productForm.category" class="input" placeholder="Starter kits" /></label>
          <label class="field"><span>Stock status</span><select v-model="productForm.stock" class="input"><option>In stock</option><option>Low stock</option><option>Out of stock</option></select></label>
          <label class="field"><span>Status</span><select v-model="productForm.status" class="input"><option value="draft">draft</option><option value="active">active</option></select></label>
          <label class="field"><span>SEO title</span><input v-model="productForm.seoTitle" class="input" /></label>
          <label class="field modal-span-2"><span>SEO description</span><textarea v-model="productForm.seoDescription" class="input textarea" rows="3"></textarea></label>
          <div class="modal-actions modal-span-2">
            <button type="button" class="button secondary" @click="closeProductModal">Cancel</button>
            <button type="submit" class="button" :disabled="ecommerce.saving">{{ ecommerce.saving ? 'Saving...' : 'Save' }}</button>
          </div>
        </form>
      </div>
    </div>

    <div v-if="showPlanModal" class="modal-backdrop" @click.self="closePlanModal">
      <div class="modal-shell">
        <div class="modal-header">
          <div>
            <p class="eyebrow">Subscription Stores</p>
            <h2>{{ planForm.id ? 'Edit plan' : 'Create plan' }}</h2>
          </div>
          <button class="icon-button" @click="closePlanModal">x</button>
        </div>
        <form class="modal-form" @submit.prevent="savePlan">
          <label class="field"><span>Plan name</span><input v-model="planForm.name" class="input" required /></label>
          <label class="field"><span>Description</span><textarea v-model="planForm.description" class="input textarea" rows="3"></textarea></label>
          <label class="field"><span>Price</span><input v-model.number="planForm.price" class="input" type="number" min="0" step="0.01" /></label>
          <label class="field"><span>Billing interval</span><select v-model="planForm.interval" class="input"><option>weekly</option><option>monthly</option><option>yearly</option></select></label>
          <label class="field"><span>Trial days</span><input v-model.number="planForm.trial" class="input" type="number" min="0" /></label>
          <label class="field"><span>Status</span><select v-model="planForm.status" class="input"><option>draft</option><option>active</option></select></label>
          <div class="modal-actions">
            <button type="button" class="button secondary" @click="closePlanModal">Cancel</button>
            <button type="submit" class="button">Save</button>
          </div>
        </form>
      </div>
    </div>

    <div v-if="showPostModal" class="modal-backdrop" @click.self="closePostModal">
      <div class="modal-shell modal-shell-wide">
        <div class="modal-header">
          <div>
            <p class="eyebrow">Blogs</p>
            <h2>{{ postForm.id ? 'Edit post' : 'New post' }}</h2>
          </div>
          <button class="icon-button" @click="closePostModal">x</button>
        </div>
        <form class="modal-form modal-form-grid" @submit.prevent="savePost">
          <label class="field"><span>Title</span><input v-model="postForm.title" class="input" required /></label>
          <label class="field"><span>Slug</span><input v-model="postForm.slug" class="input" required /></label>
          <label class="field modal-span-2"><span>Excerpt</span><textarea v-model="postForm.excerpt" class="input textarea" rows="3"></textarea></label>
          <label class="field modal-span-2"><span>Content</span><textarea v-model="postForm.content" class="input textarea" rows="8"></textarea></label>
          <label class="field"><span>Featured image URL</span><input v-model="postForm.featuredImage" class="input" /></label>
          <label class="field"><span>Tags</span><input v-model="postForm.tags" class="input" placeholder="launch, product" /></label>
          <label class="field"><span>Author</span><input v-model="postForm.author" class="input" /></label>
          <label class="field"><span>Visibility</span><select v-model="postForm.visibility" class="input"><option>public</option><option>members</option><option>paid</option></select></label>
          <label class="field"><span>Status</span><select v-model="postForm.status" class="input"><option value="draft">draft</option><option value="published">published</option></select></label>
          <label class="field"><span>SEO title</span><input v-model="postForm.seoTitle" class="input" /></label>
          <label class="field modal-span-2"><span>SEO description</span><textarea v-model="postForm.seoDescription" class="input textarea" rows="3"></textarea></label>
          <div class="modal-actions modal-span-2">
            <button type="button" class="button secondary" @click="closePostModal">Cancel</button>
            <button type="submit" class="button" :disabled="blogs.saving">{{ blogs.saving ? 'Saving...' : 'Save' }}</button>
          </div>
        </form>
      </div>
    </div>
  </main>
</template>

<script setup>
import { computed, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { api } from '../services/api'
import { useEcommerceStore } from '../stores/ecommerce'
import { useBlogsStore } from '../stores/blogs'
import { useSubscriptionsStore } from '../stores/subscriptions'

const route = useRoute()
const router = useRouter()
const site = ref(null)
const activeSection = ref('overview')
const ecommerceTab = ref('Products')
const subscriptionTab = ref('Plans')
const blogsTab = ref('Posts')
const showProductModal = ref(false)
const showPlanModal = ref(false)
const showPostModal = ref(false)
const ecommerce = useEcommerceStore()
const blogs = useBlogsStore()
const subscriptionsStore = useSubscriptionsStore()

const blankProduct = () => ({ id: null, name: '', slug: '', description: '', price: 0, imageUrl: '', status: 'draft', sku: '', category: '', stock: 'In stock', seoTitle: '', seoDescription: '' })
const blankPlan = () => ({ id: null, name: '', description: '', price: 0, interval: 'monthly', trial: 14, status: 'draft' })
const blankPost = () => ({ id: null, title: '', slug: '', excerpt: '', content: '', status: 'draft', featuredImage: '', tags: '', author: 'Ava Chen', seoTitle: '', seoDescription: '', visibility: 'public' })

const productForm = ref(blankProduct())
const planForm = ref(blankPlan())
const postForm = ref(blankPost())

const sections = [
  { key: 'overview', label: 'Overview' },
  { key: 'analytics', label: 'Analytics' },
  { key: 'pages', label: 'Pages' },
  { key: 'ecommerce', label: 'Ecommerce' },
  { key: 'subscriptions', label: 'Subscription Stores' },
  { key: 'blogs', label: 'Blogs' },
  { key: 'deployments', label: 'Deployments' },
  { key: 'settings', label: 'Settings' }
]

const ecommerceTabs = ['Products', 'Orders', 'Customers', 'Coupons', 'Shipping', 'Taxes', 'Store Settings']
const subscriptionTabs = ['Plans', 'Subscribers', 'Subscriptions', 'Invoices', 'Failed Payments', 'Settings']
const blogTabs = ['Posts', 'Categories', 'Tags', 'Authors', 'Newsletter', 'Members', 'Analytics', 'Settings']

const storeSettingCards = computed(() => [
  { eyebrow: 'Currency', title: ecommerce.storeSettings?.currency || 'USD', lead: 'Storefront currency and display format.' },
  { eyebrow: 'Store email', title: ecommerce.storeSettings?.storeEmail || 'orders@sitecraft.app', lead: 'Primary inbox for order notifications.' },
  { eyebrow: 'Checkout mode', title: ecommerce.storeSettings?.checkoutMode || 'standard', lead: 'Checkout flow mode and readiness.' },
  { eyebrow: 'Payment provider', title: ecommerce.storeSettings?.paymentProvider || 'coming_soon', lead: 'Payment provider placeholder.' }
])
const subscriptionSettingCards = computed(() => [
  { eyebrow: 'Billing currency', title: subscriptionsStore.settings?.billingCurrency || 'USD', lead: 'Set the default recurring billing currency.' },
  { eyebrow: 'Renewal emails', title: subscriptionsStore.settings?.renewalEmailEnabled ? 'Enabled' : 'Disabled', lead: 'Send renewal reminders and receipt notifications.' },
  { eyebrow: 'Cancellation policy', title: subscriptionsStore.settings?.cancellationPolicy || 'Cancel before renewal', lead: 'Allow cancellations before the next billing cycle.' },
  { eyebrow: 'Payment provider', title: 'Coming soon', lead: 'Billing provider connection placeholder.' }
])
const blogAnalyticsCards = computed(() => [
  { eyebrow: 'Views', title: `${blogs.analytics?.views || 0}`, lead: 'Total views across current posts and content.' },
  { eyebrow: 'Top posts', title: `${blogs.posts[0]?.title || 'No posts yet'}`, lead: 'Current leading article by reads.' },
  { eyebrow: 'Newsletter', title: `${blogs.analytics?.newsletterOpens || 0}`, lead: 'Newsletter opens tracked for the current site.' },
  { eyebrow: 'Members', title: `${blogs.analytics?.signups || 0}`, lead: 'Signups attributed to blog content.' }
])
const blogSettingCards = computed(() => [
  { eyebrow: 'Blog title', title: `${site.value?.name || 'SiteCraft'} Journal`, lead: 'The public title for the blog workspace.' },
  { eyebrow: 'Default author', title: blogs.authors[0]?.name || 'Ava Chen', lead: 'Used for new posts by default.' },
  { eyebrow: 'RSS', title: 'Enabled', lead: 'RSS feed is ready for readers and syndication.' },
  { eyebrow: 'Canonical & sharing', title: 'Coming soon', lead: 'Social sharing and canonical preferences placeholder.' }
])

const sectionContent = {
  overview: {
    eyebrow: 'Site overview',
    title: computed(() => site.value?.name || 'Project dashboard'),
    lead: computed(() => site.value?.description || 'View performance, analytics, and publication details for this site.'),
    metrics: ['18.4k', '2.8%']
  },
  analytics: {
    eyebrow: 'Analytics',
    title: 'Traffic and engagement',
    lead: 'Track traffic quality, top channels, and what is converting across the site.',
    metrics: ['18.4k', '92%']
  },
  pages: {
    eyebrow: 'Pages',
    title: 'Page inventory',
    lead: 'Manage published routes, drafts, and page operations before you open the editor.',
    metrics: ['12', '4 live']
  },
  deployments: {
    eyebrow: 'Deployments',
    title: 'Release activity',
    lead: 'See preview and production publish history for the current site.',
    metrics: ['7', 'Live']
  },
  settings: {
    eyebrow: 'Settings',
    title: 'Project settings',
    lead: 'Adjust naming, permissions, integrations, and site-level preferences.',
    metrics: ['1', 'OK']
  }
}

const sectionMeta = computed(() => sectionContent[activeSection.value] || sectionContent.overview)
const sectionCards = computed(() => {
  const map = {
    overview: [
      { eyebrow: 'Next action', title: 'Keep building', lead: 'Jump back into the editor or publish the latest version.', accent: true },
      { eyebrow: 'Commerce', title: 'Launch products', lead: 'Set up products, discounts, shipping, and store settings.' },
      { eyebrow: 'Publishing', title: 'Grow content', lead: 'Write posts, start a newsletter, and prepare members.' }
    ],
    analytics: [
      { eyebrow: 'Traffic', title: 'Sessions trend', lead: 'Weekly sessions and returning visitor quality.', accent: true },
      { eyebrow: 'Sources', title: 'Top channels', lead: 'Search, direct, referral, and paid source placeholders.' },
      { eyebrow: 'Performance', title: 'Conversion signals', lead: 'Which pages and offers are performing best.' }
    ],
    pages: [
      { eyebrow: 'Routes', title: 'Published pages', lead: 'Review live pages and what is still in draft.', accent: true },
      { eyebrow: 'Operations', title: 'Editing flow', lead: 'Open the editor to refine structure, copy, and layout.' },
      { eyebrow: 'Readiness', title: 'Publish prep', lead: 'Check pages before the next production publish.' }
    ],
    deployments: [
      { eyebrow: 'Production', title: 'Latest publish', lead: 'Most recent production release and health status.', accent: true },
      { eyebrow: 'History', title: 'Release log', lead: 'Preview and production publish history for the team.' },
      { eyebrow: 'Recovery', title: 'Rollback readiness', lead: 'Operational placeholders for future deployment controls.' }
    ],
    settings: [
      { eyebrow: 'Branding', title: 'General settings', lead: 'Name, metadata, and environment-level preferences.', accent: true },
      { eyebrow: 'Access', title: 'Permissions', lead: 'Keep collaborators and roles organized.' },
      { eyebrow: 'Safety', title: 'Critical actions', lead: 'Archive or hand off the project when needed.' }
    ]
  }
  return map[activeSection.value] || map.overview
})

const ecommerceProducts = computed(() => ecommerce.products.map((product, index) => ({
  ...product,
  sku: product.sku || `SC-${String(index + 1).padStart(3, '0')}`,
  category: product.category || 'General',
  stock: product.stock || (product.status === 'active' ? 'In stock' : 'Low stock'),
  seoTitle: product.seoTitle || product.name,
  seoDescription: product.seoDescription || product.description
})))

const blogPosts = computed(() => blogs.posts.map((post) => ({
  ...post,
  author: post.author || 'Ava Chen',
  featuredImage: post.featuredImage || '',
  tags: post.tags || '',
  seoTitle: post.seoTitle || post.title,
  seoDescription: post.seoDescription || post.excerpt,
  visibility: post.visibility || 'public'
})))
const orderRows = computed(() => ecommerce.orders)
const customerRows = computed(() => ecommerce.customers)
const couponRows = computed(() => ecommerce.coupons)
const shippingZoneCards = computed(() => (ecommerce.shippingZones.length ? ecommerce.shippingZones.map((zone) => ({
  eyebrow: zone.region || 'Shipping zone',
  title: zone.name,
  lead: `${zone.method} at ${currency(zone.rate)}`
})) : [{ eyebrow: 'Shipping zones', title: 'No shipping zones yet', lead: 'Add your first shipping zone to start configuring delivery rules.' }]))
const subscriberRows = computed(() => subscriptionsStore.subscribers)
const subscriptionRows = computed(() => subscriptionsStore.subscriptions)
const invoiceRows = computed(() => subscriptionsStore.invoices)
const failedPaymentRows = computed(() => subscriptionsStore.failedPayments)

const stamp = (value) => value ? new Date(value).toLocaleDateString() : 'Just now'
const currency = (value) => new Intl.NumberFormat('en-US', { style: 'currency', currency: 'USD' }).format(Number(value || 0))

const loadSite = async () => {
  const { data } = await api.get(`/projects/${route.params.id}`)
  site.value = data?.item || data || null
}

const loadSectionData = async (section) => {
  if (section === 'ecommerce') {
    await ecommerce.load(route.params.id)
  }
  if (section === 'blogs') {
    await blogs.load(route.params.id)
  }
  if (section === 'subscriptions') {
    await subscriptionsStore.load(route.params.id)
  }
}

const openProductModal = (product = null) => {
  productForm.value = product ? { ...product } : blankProduct()
  showProductModal.value = true
}
const closeProductModal = () => {
  productForm.value = blankProduct()
  showProductModal.value = false
}
const saveProduct = async () => {
  await ecommerce.save(route.params.id, productForm.value)
  if (!ecommerce.error) closeProductModal()
}
const duplicateProduct = async (id) => {
  await ecommerce.duplicate(route.params.id, id)
}
const removeProduct = async (id) => {
  if (window.confirm('Delete this product?')) {
    await ecommerce.remove(route.params.id, id)
  }
}

const openPlanModal = (plan = null) => {
  planForm.value = plan ? { ...plan } : blankPlan()
  showPlanModal.value = true
}
const closePlanModal = () => {
  planForm.value = blankPlan()
  showPlanModal.value = false
}
const savePlan = () => {
  subscriptionsStore.savePlan(route.params.id, planForm.value).then(closePlanModal)
}
const removePlan = (id) => {
  if (window.confirm('Delete this plan?')) {
    subscriptionsStore.removePlan(route.params.id, id)
  }
}

const openPostModal = (post = null) => {
  postForm.value = post ? { ...post } : blankPost()
  showPostModal.value = true
}
const closePostModal = () => {
  postForm.value = blankPost()
  showPostModal.value = false
}
const savePost = async () => {
  await blogs.save(route.params.id, postForm.value)
  if (!blogs.error) closePostModal()
}
const removePost = async (id) => {
  if (window.confirm('Delete this post?')) {
    await blogs.remove(route.params.id, id)
  }
}
const togglePost = async (post) => {
  await blogs.setPublished(route.params.id, post.id, post.status !== 'published')
}

const backToWorkspace = async () => {
  await router.push('/dashboard')
}
const editSite = async () => {
  await router.push(`/site/${route.params.id}/editor`)
}
const signOut = async () => {
  localStorage.removeItem('sitecraft_token')
  await router.push('/login')
}

watch(activeSection, (section) => {
  loadSectionData(section)
})

onMounted(async () => {
  await loadSite()
  await loadSectionData(activeSection.value)
})
</script>
