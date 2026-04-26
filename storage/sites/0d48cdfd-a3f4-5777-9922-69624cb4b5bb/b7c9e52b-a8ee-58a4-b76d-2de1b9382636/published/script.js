const CART_KEY = "sitecraft-demo-cart"

function readCart() {
  try {
    const cart = JSON.parse(window.localStorage.getItem(CART_KEY) || "[]")
    return Array.isArray(cart) ? cart : []
  } catch {
    return []
  }
}

function writeCart(cart) {
  window.localStorage.setItem(CART_KEY, JSON.stringify(cart))
}

function formatMoney(value) {
  const number = Number(value || 0)
  return `INR ${number.toFixed(2)}`
}

function getRenderedProducts() {
  return [...document.querySelectorAll("#shop-grid .product-card")].map((card, index) => ({
    id: card.dataset.slug || card.dataset.name || `product-${index}`,
    name: card.dataset.name || "",
    category: card.dataset.category || "General",
    price: Number(card.dataset.price || 0),
    description: card.dataset.description || "",
    status: card.dataset.status || "",
  }))
}

function updateCartCount() {
  const count = readCart().reduce((sum, item) => sum + item.quantity, 0)
  document.querySelectorAll("[data-cart-count]").forEach((node) => {
    node.textContent = String(count)
  })
}

function openCart() {
  const drawer = document.getElementById("cart-drawer")
  const overlay = document.getElementById("cart-overlay")
  if (!drawer || !overlay) return
  drawer.classList.add("is-open")
  drawer.setAttribute("aria-hidden", "false")
  overlay.hidden = false
}

function closeCart() {
  const drawer = document.getElementById("cart-drawer")
  const overlay = document.getElementById("cart-overlay")
  if (!drawer || !overlay) return
  drawer.classList.remove("is-open")
  drawer.setAttribute("aria-hidden", "true")
  overlay.hidden = true
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
  updateCartCount()
  renderCart()
  openCart()
}

function changeQuantity(productId, delta) {
  const next = readCart()
    .map((item) => (item.id === productId ? { ...item, quantity: item.quantity + delta } : item))
    .filter((item) => item.quantity > 0)
  writeCart(next)
  updateCartCount()
  renderCart()
}

function removeFromCart(productId) {
  writeCart(readCart().filter((item) => item.id !== productId))
  updateCartCount()
  renderCart()
}

function renderCart() {
  const itemsNode = document.getElementById("cart-items")
  const emptyNode = document.getElementById("cart-empty")
  const subtotalNode = document.getElementById("cart-subtotal")
  if (!itemsNode || !emptyNode || !subtotalNode) return

  const products = getRenderedProducts()
  const cart = readCart()
  itemsNode.innerHTML = ""

  if (!cart.length) {
    emptyNode.hidden = false
    itemsNode.hidden = true
    subtotalNode.textContent = formatMoney(0)
    return
  }

  let subtotal = 0
  for (const item of cart) {
    const product = products.find((entry) => entry.id === item.id)
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
          <button type="button" data-qty="${product.id}" data-delta="-1">-</button>
          <span>${item.quantity}</span>
          <button type="button" data-qty="${product.id}" data-delta="1">+</button>
        </div>
        <button class="cart-remove" type="button" data-remove="${product.id}">Remove</button>
      </div>
    `
    itemsNode.appendChild(row)
  }

  emptyNode.hidden = subtotal > 0
  itemsNode.hidden = subtotal === 0
  subtotalNode.textContent = formatMoney(subtotal)

  itemsNode.querySelectorAll("[data-qty]").forEach((button) => {
    button.addEventListener("click", () => {
      changeQuantity(button.getAttribute("data-qty"), Number(button.getAttribute("data-delta") || 0))
    })
  })
  itemsNode.querySelectorAll("[data-remove]").forEach((button) => {
    button.addEventListener("click", () => removeFromCart(button.getAttribute("data-remove")))
  })
}

function initShopPage() {
  const grid = document.getElementById("shop-grid")
  if (!grid) return

  const search = document.getElementById("shop-search")
  const category = document.getElementById("shop-category")
  const sort = document.getElementById("shop-sort")
  const empty = document.getElementById("shop-empty")
  const resultsMeta = document.getElementById("shop-results-meta")
  const reset = document.getElementById("shop-reset")

  const products = getRenderedProducts()
  const categories = [...new Set(products.map((product) => product.category).filter(Boolean))]
  categories.forEach((name) => {
    const option = document.createElement("option")
    option.value = name
    option.textContent = name
    category.appendChild(option)
  })

  const bindAddButtons = () => {
    grid.querySelectorAll(".add-to-cart").forEach((button) => {
      button.addEventListener("click", () => {
        const card = button.closest(".product-card")
        if (!card) return
        addToCart(card.dataset.slug || card.dataset.name)
      })
    })
  }

  const update = () => {
    const term = search.value.trim().toLowerCase()
    const selectedCategory = category.value
    const sortValue = sort.value
    const cards = [...grid.querySelectorAll(".product-card")]

    let visible = cards.filter((card) => {
      const name = (card.dataset.name || "").toLowerCase()
      const description = (card.dataset.description || "").toLowerCase()
      const cardCategory = card.dataset.category || ""
      const searchMatch = !term || name.includes(term) || description.includes(term) || cardCategory.toLowerCase().includes(term)
      const categoryMatch = selectedCategory === "all" || cardCategory === selectedCategory
      return searchMatch && categoryMatch
    })

    if (sortValue !== "featured") {
      visible = [...visible].sort((left, right) => {
        if (sortValue === "price-asc") return Number(left.dataset.price || 0) - Number(right.dataset.price || 0)
        if (sortValue === "price-desc") return Number(right.dataset.price || 0) - Number(left.dataset.price || 0)
        return (left.dataset.name || "").localeCompare(right.dataset.name || "")
      })
      visible.forEach((card) => grid.appendChild(card))
    }

    cards.forEach((card) => {
      card.hidden = !visible.includes(card)
    })

    empty.hidden = visible.length > 0
    resultsMeta.textContent = `${visible.length} product${visible.length === 1 ? "" : "s"} shown`
  }

  ;[search, category, sort].forEach((node) => node.addEventListener("input", update))
  reset.addEventListener("click", () => {
    search.value = ""
    category.value = "all"
    sort.value = "featured"
    update()
  })

  bindAddButtons()
  update()
}

function initBlogSearch() {
  const search = document.getElementById("blog-search")
  const grid = document.getElementById("blog-grid")
  const empty = document.getElementById("blog-empty")
  if (!search || !grid || !empty) return

  const cards = [...grid.querySelectorAll(".post-card")]
  search.addEventListener("input", () => {
    const term = search.value.trim().toLowerCase()
    const visible = cards.filter((card) => {
      const title = (card.dataset.title || "").toLowerCase()
      const author = (card.dataset.author || "").toLowerCase()
      return !term || title.includes(term) || author.includes(term)
    })
    cards.forEach((card) => {
      card.hidden = !visible.includes(card)
    })
    empty.hidden = visible.length > 0
  })
}

function initContactForm() {
  const form = document.getElementById("contact-form")
  const message = document.getElementById("contact-message")
  if (!form || !message) return
  form.addEventListener("submit", (event) => {
    event.preventDefault()
    message.hidden = false
    form.querySelector("button[type='submit']").textContent = "Submitted"
  })
}

function initNewsletterForm() {
  const form = document.getElementById("newsletter-form")
  const message = document.getElementById("newsletter-message")
  if (!form || !message) return
  form.addEventListener("submit", (event) => {
    event.preventDefault()
    message.hidden = false
    form.querySelector("button[type='submit']").textContent = "Joined"
  })
}

function initNav() {
  const currentPath = window.location.pathname.split("/").pop() || "index.html"
  document.querySelectorAll(".nav-link").forEach((link) => {
    if (link.getAttribute("href") === currentPath || (currentPath === "" && link.getAttribute("href") === "index.html")) {
      link.setAttribute("aria-current", "page")
    }
  })

  const toggle = document.querySelector(".nav-toggle")
  const nav = document.getElementById("primary-nav")
  if (toggle && nav) {
    toggle.addEventListener("click", () => {
      const open = nav.classList.toggle("is-open")
      toggle.setAttribute("aria-expanded", open ? "true" : "false")
    })
  }
}

function initCartTriggers() {
  document.querySelectorAll("[data-open-cart]").forEach((button) => {
    button.addEventListener("click", () => {
      if (document.body.dataset.page === "shop") {
        openCart()
      } else {
        window.location.href = "shop.html"
      }
    })
  })
  const close = document.getElementById("cart-close")
  const overlay = document.getElementById("cart-overlay")
  if (close) close.addEventListener("click", closeCart)
  if (overlay) overlay.addEventListener("click", closeCart)
}

document.addEventListener("DOMContentLoaded", () => {
  initNav()
  initCartTriggers()
  initShopPage()
  initBlogSearch()
  initContactForm()
  initNewsletterForm()
  updateCartCount()
  renderCart()
})
