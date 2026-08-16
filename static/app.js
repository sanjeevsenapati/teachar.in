/**
 * TEACHAR.in - Interactive Client & Cart App JS
 */

document.addEventListener('DOMContentLoaded', () => {
    initActiveNav();
    initCartDrawer();
    initCategoryTabs();
    initSearchFilter();
    initAddToCartButtons();
});

function initActiveNav() {
    const currentPath = window.location.pathname;
    const navLinks = document.querySelectorAll('.nav-link');
    navLinks.forEach(link => {
        if (link.getAttribute('href') === currentPath) {
            link.classList.add('active');
        }
    });
}

let cart = JSON.parse(localStorage.getItem('teachar_cart') || '[]');

function saveCart() {
    localStorage.setItem('teachar_cart', JSON.stringify(cart));
    updateCartUI();
}

function updateCartUI() {
    const badgeCount = document.getElementById('cart-badge-count');
    const itemsContainer = document.getElementById('cart-items-container');
    const subtotalEl = document.getElementById('cart-subtotal');
    const taxEl = document.getElementById('cart-tax');
    const totalEl = document.getElementById('cart-total');
    const checkoutBtn = document.getElementById('checkout-btn');

    const totalItemCount = cart.reduce((sum, item) => sum + item.quantity, 0);
    if (badgeCount) badgeCount.textContent = totalItemCount;

    if (!itemsContainer) return;

    if (cart.length === 0) {
        itemsContainer.innerHTML = `
            <div class="cart-empty-state">
                <i class="fa-solid fa-utensils"></i>
                <p>Your cart is empty</p>
                <a href="/menu" class="btn btn-secondary btn-sm" id="cart-explore-btn">Explore Menu</a>
            </div>
        `;
        if (subtotalEl) subtotalEl.textContent = '₹0';
        if (taxEl) taxEl.textContent = '₹0';
        if (totalEl) totalEl.textContent = '₹0';
        if (checkoutBtn) checkoutBtn.disabled = true;
        
        const exploreBtn = document.getElementById('cart-explore-btn');
        if (exploreBtn) exploreBtn.addEventListener('click', closeCart);
        return;
    }

    let subtotal = 0;
    let itemsHTML = '';

    cart.forEach(item => {
        const itemTotal = item.price * item.quantity;
        subtotal += itemTotal;

        itemsHTML += `
            <div class="cart-item" data-id="${item.id}">
                <img src="${item.image || '/static/images/hero-banner.jpg'}" alt="${item.name}" class="cart-item-img">
                <div class="cart-item-details">
                    <div class="cart-item-title">${item.name}</div>
                    <div class="cart-item-price">₹${item.price}</div>
                    <div class="cart-item-qty">
                        <button class="qty-btn qty-minus" onclick="changeQuantity(${item.id}, -1)">-</button>
                        <span>${item.quantity}</span>
                        <button class="qty-btn qty-plus" onclick="changeQuantity(${item.id}, 1)">+</button>
                    </div>
                </div>
                <button class="cart-item-remove" onclick="removeFromCart(${item.id})" style="background:none;border:none;color:#9c8a7f;cursor:pointer;">
                    <i class="fa-solid fa-trash-can"></i>
                </button>
            </div>
        `;
    });

    itemsContainer.innerHTML = itemsHTML;

    const tax = Math.round(subtotal * 0.05);
    const total = subtotal + tax;

    if (subtotalEl) subtotalEl.textContent = `₹${subtotal}`;
    if (taxEl) taxEl.textContent = `₹${tax}`;
    if (totalEl) totalEl.textContent = `₹${total}`;
    if (checkoutBtn) checkoutBtn.disabled = false;
}

function addToCart(item) {
    const existing = cart.find(i => i.id === item.id);
    if (existing) {
        existing.quantity += 1;
    } else {
        cart.push({ ...item, quantity: 1 });
    }
    saveCart();
    showToast(`Added ${item.name} to cart!`);
}

window.changeQuantity = function(id, delta) {
    const item = cart.find(i => i.id === id);
    if (item) {
        item.quantity += delta;
        if (item.quantity <= 0) {
            cart = cart.filter(i => i.id !== id);
        }
        saveCart();
    }
};

window.removeFromCart = function(id) {
    cart = cart.filter(i => i.id !== id);
    saveCart();
    showToast('Item removed from cart');
};

function initCartDrawer() {
    const openBtn = document.getElementById('open-cart-btn');
    const closeBtn = document.getElementById('close-cart-btn');
    const overlay = document.getElementById('cart-overlay');
    const checkoutBtn = document.getElementById('checkout-btn');

    if (openBtn) openBtn.addEventListener('click', openCart);
    if (closeBtn) closeBtn.addEventListener('click', closeCart);
    if (overlay) overlay.addEventListener('click', closeCart);

    if (checkoutBtn) {
        checkoutBtn.addEventListener('click', async () => {
            if (cart.length === 0) return;

            checkoutBtn.disabled = true;
            checkoutBtn.innerHTML = '<span>Processing...</span> <i class="fa-solid fa-spinner fa-spin"></i>';

            try {
                const response = await fetch('/api/orders', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({
                        items: cart.map(item => ({
                            id: item.id,
                            name: item.name,
                            price: item.price,
                            quantity: item.quantity
                        }))
                    })
                });

                if (response.ok) {
                    cart = [];
                    saveCart();
                    closeCart();
                    showToast('🎉 Order placed successfully!');
                    setTimeout(() => {
                        window.location.href = '/orders';
                    }, 1200);
                } else {
                    showToast('Failed to place order. Please try again.');
                }
            } catch (err) {
                console.error(err);
                showToast('An error occurred. Please try again.');
            } finally {
                checkoutBtn.disabled = false;
                checkoutBtn.innerHTML = '<span>Place Order</span> <i class="fa-solid fa-check"></i>';
            }
        });
    }

    updateCartUI();
}

function openCart() {
    const overlay = document.getElementById('cart-overlay');
    const drawer = document.getElementById('cart-drawer');
    if (overlay) overlay.classList.add('active');
    if (drawer) drawer.classList.add('active');
}

function closeCart() {
    const overlay = document.getElementById('cart-overlay');
    const drawer = document.getElementById('cart-drawer');
    if (overlay) overlay.classList.remove('active');
    if (drawer) drawer.classList.remove('active');
}

function initAddToCartButtons() {
    document.addEventListener('click', (e) => {
        const btn = e.target.closest('.add-to-cart-btn');
        if (btn) {
            const id = parseInt(btn.dataset.id, 10);
            const name = btn.dataset.name;
            const price = parseFloat(btn.dataset.price);
            const image = btn.dataset.image;

            addToCart({ id, name, price, image });
        }
    });
}

function initCategoryTabs() {
    const tabs = document.querySelectorAll('.tab-btn');
    if (!tabs.length) return;

    tabs.forEach(tab => {
        tab.addEventListener('click', () => {
            tabs.forEach(t => t.classList.remove('active'));
            tab.classList.add('active');
            filterMenu(tab.dataset.category, getSearchTerm());
        });
    });
}

function initSearchFilter() {
    const searchInput = document.getElementById('menu-search-input');
    if (!searchInput) return;

    searchInput.addEventListener('input', (e) => {
        filterMenu(getActiveCategory(), e.target.value.toLowerCase().trim());
    });
}

function getActiveCategory() {
    const activeTab = document.querySelector('.tab-btn.active');
    return activeTab ? activeTab.dataset.category : 'all';
}

function getSearchTerm() {
    const searchInput = document.getElementById('menu-search-input');
    return searchInput ? searchInput.value.toLowerCase().trim() : '';
}

function filterMenu(category, searchTerm) {
    const categoryBlocks = document.querySelectorAll('.menu-category-block');
    const cards = document.querySelectorAll('.menu-card');
    const noResultsEl = document.getElementById('no-search-results');

    let visibleCardsCount = 0;

    cards.forEach(card => {
        const itemCategory = card.dataset.category;
        const itemName = card.dataset.name.toLowerCase();

        const matchesCategory = (category === 'all' || itemCategory === category);
        const matchesSearch = (!searchTerm || itemName.includes(searchTerm));

        if (matchesCategory && matchesSearch) {
            card.classList.remove('hidden');
            visibleCardsCount++;
        } else {
            card.classList.add('hidden');
        }
    });

    categoryBlocks.forEach(block => {
        const visibleCardsInBlock = block.querySelectorAll('.menu-card:not(.hidden)');
        if (visibleCardsInBlock.length === 0) {
            block.classList.add('hidden');
        } else {
            block.classList.remove('hidden');
        }
    });

    if (noResultsEl) {
        if (visibleCardsCount === 0) {
            noResultsEl.classList.remove('hidden');
        } else {
            noResultsEl.classList.add('hidden');
        }
    }
}

function showToast(message) {
    const toast = document.getElementById('toast');
    const msgEl = document.getElementById('toast-message');
    if (!toast || !msgEl) return;

    msgEl.textContent = message;
    toast.classList.add('active');

    setTimeout(() => {
        toast.classList.remove('active');
    }, 3000);
}