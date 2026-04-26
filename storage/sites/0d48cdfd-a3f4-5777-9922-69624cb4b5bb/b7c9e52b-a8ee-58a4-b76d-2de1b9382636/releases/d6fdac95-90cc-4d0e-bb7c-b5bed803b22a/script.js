const SHOP_PRODUCTS = [
  { id: "strategy-toolkit", name: "Strategy Toolkit", category: "Strategy", price: 149, description: "Planning templates for offer refinement, positioning, and leadership workshops.", badge: "Best seller" },
  { id: "automation-blueprint", name: "Automation Blueprint", category: "Operations", price: 249, description: "Workflow map and automation recommendations for service delivery teams.", badge: "Popular" },
  { id: "executive-kpi-dashboard", name: "Executive KPI Dashboard", category: "Analytics", price: 199, description: "A clean scorecard setup for tracking pipeline, delivery, and growth metrics.", badge: "New" },
  { id: "client-journey-pack", name: "Client Journey Pack", category: "Experience", price: 129, description: "Journey mapping kit for onboarding, communication, and retention touchpoints.", badge: "Limited" },
  { id: "website-positioning-kit", name: "Website Positioning Kit", category: "Website", price: 179, description: "Page messaging prompts, proof-point blocks, and CTA structure examples.", badge: "Featured" },
  { id: "offer-audit-session", name: "Offer Audit Session", category: "Strategy", price: 299, description: "A sample consulting engagement pack for auditing service offers and pricing narratives.", badge: "Premium" },
  { id: "ops-review-template", name: "Ops Review Template", category: "Operations", price: 89, description: "A lightweight review worksheet for surfacing bottlenecks and handoff gaps.", badge: "Starter" },
  { id: "lead-qualifier-kit", name: "Lead Qualifier Kit", category: "Website", price: 159, description: "Form logic and qualification prompts to help teams attract higher-fit inquiries.", badge: "Recommended" },
]

const CART_KEY = "sitecraft-demo-cart"

function readCart() {
  try {
    const parsed = JSON.parse(window.localStorage.getItem(CART_KEY) || "[]")
    return Array.isArray(parsed) ? parsed : []
  } catch {
    return []
  }
}

function writeCart(cart) {
  window.localStorage.setItem(CART_KEY, JSON.stringify(cart))
}

function formatMoney(value) {
  return new Intl.NumberFormat("en-US", { style: "currency", currency: "USD", maximumFractionDigits: 0 }).format(value)
}

function updateGlobalCartCount() {
  const count = readCart().reduce((sum, item) => sum + item.quantity, 0)
  document.querySelectorAll("[data-cart-count]").forEach((node) => {
    node.textContent = String(count)
  })
}

function ensureNavCartButton() {
  const navBar = document.querySelector(".nav-bar")
  if (!navBar || navBar.querySelector(".nav-cart-link")) return
  const button = document.createElement("button")
  button.type = "button"
  button.className = "button button-secondary nav-cart-link"
  button.innerHTML = 'Cart <span class="cart-badge" data-cart-count>0</span>'
  button.addEventListener("click", () => {
    if (document.body.dataset.page === "shop") {
      openCart()
    } else {
      window.location.href = "shop.html"
    }
  })
  navBar.appendChild(button)
}

function addToCart(productId) {
  const cart = readCart()
  const existing = cart.find((item) => item.id === productId)
  if (existing) {
    existing.quantity += 1
  } else {
    cart.push({ id: productId, quantity: 1 })
  }
  writeCart(cart)
  updateGlobalCartCount()
  renderCart()
  openCart()
}

function changeQuantity(productId, delta) {
  const cart = readCart()
  const next = cart
    .map((item) => (item.id === productId ? { ...item, quantity: item.quantity + delta } : item))
    .filter((item) => item.quantity > 0)
  writeCart(next)
  updateGlobalCartCount()
  renderCart()
}

function removeFromCart(productId) {
  const next = readCart().filter((item) => item.id !== productId)
  writeCart(next)
  updateGlobalCartCount()
  renderCart()
}

function openCart() {
  const drawer = document.getElementById("cart-drawer")
  const overlay = document.getElementById("cart-overlay")
  if (!drawer || !overlay) return
  drawer.classList.add("is-open")
  drawer.setAttribute("aria-hidden", "false")
  overlay.hidden = false
  overlay.classList.add("is-visible")
}

function closeCart() {
  const drawer = document.getElementById("cart-drawer")
  const overlay = document.getElementById("cart-overlay")
  if (!drawer || !overlay) return
  drawer.classList.remove("is-open")
  drawer.setAttribute("aria-hidden", "true")
  overlay.classList.remove("is-visible")
  overlay.hidden = true
}

function renderCart() {
  const cartItems = document.getElementById("cart-items")
  const empty = document.getElementById("cart-empty")
  const subtotalNode = document.getElementById("cart-subtotal")
  if (!cartItems || !empty || !subtotalNode) return

  const cart = readCart()
  cartItems.innerHTML = ""

  if (!cart.length) {
    empty.hidden = false
    cartItems.hidden = true
    subtotalNode.textContent = formatMoney(0)
    return
  }

  empty.hidden = true
  cartItems.hidden = false

  let subtotal = 0
  for (const item of cart) {
    const product = SHOP_PRODUCTS.find((entry) => entry.id === item.id)
    if (!product) continue
    subtotal += product.price * item.quantity

    const row = document.createElement("article")
    row.className = "cart-item"
    row.innerHTML = `
      <div class="cart-item__body">
        <p class="card-kicker">${product.category}</p>
        <h3>${product.name}</h3>
        <p>${product.description}</p>
      </div>
      <div class="cart-item__meta">
        <strong>${formatMoney(product.price)}</strong>
        <div class="qty-control">
          <button type="button" data-qty-action="decrease" data-product-id="${product.id}">-</button>
          <span>${item.quantity}</span>
          <button type="button" data-qty-action="increase" data-product-id="${product.id}">+</button>
        </div>
        <button type="button" class="cart-remove" data-remove-id="${product.id}">Remove</button>
      </div>
    `
    cartItems.appendChild(row)
  }

  subtotalNode.textContent = formatMoney(subtotal)

  cartItems.querySelectorAll("[data-qty-action]").forEach((button) => {
    button.addEventListener("click", () => {
      const id = button.getAttribute("data-product-id")
      const action = button.getAttribute("data-qty-action")
      changeQuantity(id, action === "increase" ? 1 : -1)
    })
  })
  cartItems.querySelectorAll("[data-remove-id]").forEach((button) => {
    button.addEventListener("click", () => removeFromCart(button.getAttribute("data-remove-id")))
  })
}

function renderShop() {
  const grid = document.getElementById("shop-grid")
  if (!grid) return

  const search = document.getElementById("shop-search")
  const category = document.getElementById("shop-category")
  const sort = document.getElementById("shop-sort")
  const price = document.getElementById("shop-price")
  const priceValue = document.getElementById("shop-price-value")
  const reset = document.getElementById("shop-reset")
  const empty = document.getElementById("shop-empty")
  const resultsMeta = document.getElementById("shop-results-meta")

  const categories = [...new Set(SHOP_PRODUCTS.map((product) => product.category))]
  categories.forEach((name) => {
    const option = document.createElement("option")
    option.value = name
    option.textContent = name
    category.appendChild(option)
  })

  const update = () => {
    const searchTerm = search.value.trim().toLowerCase()
    const selectedCategory = category.value
    const sortValue = sort.value
    const maxPrice = Number(price.value)

    priceValue.textContent = formatMoney(maxPrice)

    let items = SHOP_PRODUCTS.filter((product) => {
      const matchesSearch =
        !searchTerm ||
        product.name.toLowerCase().includes(searchTerm) ||
        product.description.toLowerCase().includes(searchTerm) ||
        product.category.toLowerCase().includes(searchTerm)
      const matchesCategory = selectedCategory === "all" || product.category === selectedCategory
      const matchesPrice = product.price <= maxPrice
      return matchesSearch && matchesCategory && matchesPrice
    })

    if (sortValue === "price-asc") items = [...items].sort((a, b) => a.price - b.price)
    if (sortValue === "price-desc") items = [...items].sort((a, b) => b.price - a.price)
    if (sortValue === "name-asc") items = [...items].sort((a, b) => a.name.localeCompare(b.name))

    grid.innerHTML = ""
    empty.hidden = items.length > 0
    resultsMeta.textContent = `${items.length} product${items.length === 1 ? "" : "s"} shown`

    for (const product of items) {
      const card = document.createElement("article")
      card.className = "content-card product-card store-product-card"
      card.innerHTML = `
        <div class="product-header">
          <p class="card-kicker">${product.category}</p>
          <span class="status-badge ${product.badge === "Best seller" || product.badge === "Popular" ? "success" : ""}">${product.badge}</span>
        </div>
        <h3>${product.name}</h3>
        <p>${product.description}</p>
        <div class="product-footer">
          <strong class="money">${formatMoney(product.price)}</strong>
          <button class="button button-primary" type="button" data-add-id="${product.id}">Add to Cart</button>
        </div>
      `
      grid.appendChild(card)
    }

    grid.querySelectorAll("[data-add-id]").forEach((button) => {
      button.addEventListener("click", () => addToCart(button.getAttribute("data-add-id")))
    })
  }

  ;[search, category, sort, price].forEach((element) => element.addEventListener("input", update))
  reset.addEventListener("click", () => {
    search.value = ""
    category.value = "all"
    sort.value = "featured"
    price.value = "500"
    update()
  })
  update()
}

document.addEventListener("DOMContentLoaded", () => {
  const page = document.body.dataset.page

  document.querySelectorAll(".nav-link").forEach((link) => {
    const href = link.getAttribute("href")
    if ((page === "home" && href === "index.html") || href === `${page}.html`) {
      link.setAttribute("aria-current", "page")
    }
  })

  ensureNavCartButton()
  updateGlobalCartCount()

  const formButton = document.querySelector(".contact-form .button")
  if (formButton) {
    formButton.addEventListener("click", () => {
      formButton.textContent = "Thanks, we will be in touch"
      formButton.setAttribute("disabled", "disabled")
    })
  }

  const cartClose = document.getElementById("cart-close")
  const cartOverlay = document.getElementById("cart-overlay")
  const cartToggle = document.getElementById("cart-toggle")
  if (cartClose) cartClose.addEventListener("click", closeCart)
  if (cartOverlay) cartOverlay.addEventListener("click", closeCart)
  if (cartToggle) cartToggle.addEventListener("click", openCart)

  if (page === "shop") {
    renderShop()
    renderCart()
  }
})
