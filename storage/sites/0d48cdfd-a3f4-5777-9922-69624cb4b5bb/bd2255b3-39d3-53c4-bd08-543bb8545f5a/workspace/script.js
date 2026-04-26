(function () {
  var body = document.body;
  var current = (location.pathname.split('/').pop() || 'index.html').toLowerCase();
  document.querySelectorAll('.site-nav a').forEach(function (link) {
    if ((link.getAttribute('href') || '').toLowerCase() === current) link.classList.add('active');
  });

  var cartKey = 'sitecraft-demo-cart-pawnest-club';
  function readCart() {
    try { return JSON.parse(localStorage.getItem(cartKey) || '[]'); } catch (e) { return []; }
  }
  function writeCart(items) { localStorage.setItem(cartKey, JSON.stringify(items)); }
  function money(value) { return 'INR ' + Number(value || 0).toFixed(2); }
  function renderCart() {
    var items = readCart();
    var count = items.reduce(function (sum, item) { return sum + item.qty; }, 0);
    var subtotal = items.reduce(function (sum, item) { return sum + (item.price * item.qty); }, 0);
    var countNode = document.getElementById('cart-count');
    if (countNode) countNode.textContent = String(count);
    var itemsHost = document.querySelector('[data-cart-items]');
    var emptyState = document.querySelector('[data-cart-empty]');
    var subtotalNode = document.querySelector('[data-cart-subtotal]');
    if (subtotalNode) subtotalNode.textContent = money(subtotal);
    if (!itemsHost || !emptyState) return;
    itemsHost.innerHTML = '';
    if (!items.length) {
      emptyState.hidden = false;
      return;
    }
    emptyState.hidden = true;
    items.forEach(function (item, index) {
      var row = document.createElement('div');
      row.className = 'cart-line';
      row.innerHTML = '<div><strong>' + item.name + '</strong><span>' + money(item.price) + '</span></div>' +
        '<div class="qty-controls"><button type="button" data-cart-minus="' + index + '">-</button><span>' + item.qty + '</span><button type="button" data-cart-plus="' + index + '">+</button><button type="button" data-cart-remove="' + index + '">Remove</button></div>';
      itemsHost.appendChild(row);
    });
  }
  function mutateCart(mutator) {
    var next = readCart();
    mutator(next);
    writeCart(next.filter(function (item) { return item.qty > 0; }));
    renderCart();
  }
  document.addEventListener('click', function (event) {
    var add = event.target.closest('[data-add-to-cart]');
    if (add) {
      mutateCart(function (items) {
        var name = add.getAttribute('data-name') || 'Item';
        var price = Number(add.getAttribute('data-price') || 0);
        var found = items.find(function (item) { return item.name === name; });
        if (found) found.qty += 1; else items.push({ name: name, price: price, qty: 1 });
      });
      var drawer = document.querySelector('[data-cart-drawer]');
      if (drawer) drawer.hidden = false;
      return;
    }
    if (event.target.closest('[data-cart-toggle]')) {
      var drawer = document.querySelector('[data-cart-drawer]');
      if (drawer) drawer.hidden = !drawer.hidden;
      return;
    }
    if (event.target.closest('[data-cart-close]')) {
      var drawer = document.querySelector('[data-cart-drawer]');
      if (drawer) drawer.hidden = true;
      return;
    }
    if (event.target.matches('[data-cart-minus]')) {
      var minusIndex = Number(event.target.getAttribute('data-cart-minus'));
      mutateCart(function (items) { if (items[minusIndex]) items[minusIndex].qty -= 1; });
      return;
    }
    if (event.target.matches('[data-cart-plus]')) {
      var plusIndex = Number(event.target.getAttribute('data-cart-plus'));
      mutateCart(function (items) { if (items[plusIndex]) items[plusIndex].qty += 1; });
      return;
    }
    if (event.target.matches('[data-cart-remove]')) {
      var removeIndex = Number(event.target.getAttribute('data-cart-remove'));
      mutateCart(function (items) { if (items[removeIndex]) items.splice(removeIndex, 1); });
      return;
    }
    if (event.target.closest('[data-fake-checkout]')) {
      alert('Checkout is a frontend-only demo in this test site.');
    }
  });
  document.querySelectorAll('[data-filter-input]').forEach(function (input) {
    input.addEventListener('input', applyFilters);
  });
  document.querySelectorAll('[data-filter-select]').forEach(function (input) {
    input.addEventListener('change', applyFilters);
  });
  function applyFilters() {
    var term = (document.querySelector('[data-filter-input]') || {}).value || '';
    var category = (document.querySelector('[data-filter-category]') || {}).value || '';
    var extra = (document.querySelector('[data-filter-extra]') || {}).value || '';
    var sort = (document.querySelector('[data-filter-sort]') || {}).value || '';
    var cards = Array.prototype.slice.call(document.querySelectorAll('[data-product-card]'));
    cards.forEach(function (card) {
      var ok = true;
      var name = (card.getAttribute('data-name') || '').toLowerCase();
      var cardCategory = card.getAttribute('data-category') || '';
      var tag = card.getAttribute('data-extra') || '';
      if (term && name.indexOf(term.toLowerCase()) === -1) ok = false;
      if (category && cardCategory !== category) ok = false;
      if (extra && tag !== extra) ok = false;
      card.hidden = !ok;
    });
    var grid = document.querySelector('[data-product-grid]');
    if (grid && sort) {
      cards.sort(function (a, b) {
        var aPrice = Number(a.getAttribute('data-price') || 0);
        var bPrice = Number(b.getAttribute('data-price') || 0);
        return sort === 'high' ? bPrice - aPrice : aPrice - bPrice;
      }).forEach(function (card) { grid.appendChild(card); });
    }
  }
  document.querySelectorAll('[data-demo-form]').forEach(function (form) {
    form.addEventListener('submit', function (event) {
      event.preventDefault();
      var target = form.querySelector('[data-form-success]');
      if (target) target.hidden = false;
      form.reset();
    });
  });
  renderCart();
  applyFilters();
})();
