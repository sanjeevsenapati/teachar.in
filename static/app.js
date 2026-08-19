/**
 * TEACHAR.in - Interactive Client, Cart & Payment Gateway JS
 */

document.addEventListener('DOMContentLoaded', () => {
    initActiveNav();
    initCartDrawer();
    initOrderTypeTabs();
    initPaymentMethodTabs();
    initCategoryTabs();
    initSearchFilter();
    initAddToCartButtons();
    initBuyNowButtons();
    initOtpModal();
    initFavorites();
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
let favorites = JSON.parse(localStorage.getItem('teachar_favorites') || '[]');
let selectedPaymentMethod = 'UPI';
let selectedOrderType = 'Dine-in';
let appliedCouponCode = '';
let discountAmount = 0;
let pendingOrderAction = null; // Stored callback executed immediately after in-modal login

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
    const paymentSection = document.getElementById('payment-options-section');
    const fulfillmentSection = document.getElementById('order-fulfillment-section');

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
        if (paymentSection) paymentSection.style.display = 'none';
        if (fulfillmentSection) fulfillmentSection.style.display = 'none';
        
        const exploreBtn = document.getElementById('cart-explore-btn');
        if (exploreBtn) exploreBtn.addEventListener('click', closeCart);
        return;
    }

    if (paymentSection) paymentSection.style.display = 'block';
    if (fulfillmentSection) fulfillmentSection.style.display = 'block';

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

    // Subscriber Discount & Automated Member Coupon Calculation
    let subscriberDiscount = 0;
    const subDiscountRow = document.getElementById('cart-subscriber-discount-row');
    const subTierNameEl = document.getElementById('cart-subscriber-tier-name');
    const subDiscountEl = document.getElementById('cart-subscriber-discount');
    const couponInput = document.getElementById('coupon-code-input');
    const couponMsgEl = document.getElementById('coupon-message');

    if (window.USER_SUBSCRIPTION && window.USER_SUBSCRIPTION.discount_percent > 0) {
        subscriberDiscount = Math.round((subtotal * window.USER_SUBSCRIPTION.discount_percent) / 100 * 100) / 100;
        const autoMemberCode = (window.USER_SUBSCRIPTION.tier_id || 'VIP').toUpperCase() + 'VIP';
        
        if (!appliedCouponCode || appliedCouponCode.endsWith('VIP')) {
            appliedCouponCode = autoMemberCode;
            if (couponInput) couponInput.value = autoMemberCode;
            if (couponMsgEl) {
                couponMsgEl.style.display = 'block';
                couponMsgEl.style.color = '#d97706';
                couponMsgEl.innerHTML = `<i class="fa-solid fa-crown me-1"></i> Auto-Applied ${window.USER_SUBSCRIPTION.tier_name} Coupon <strong>${autoMemberCode}</strong> (${window.USER_SUBSCRIPTION.discount_percent}% OFF)`;
            }
        }

        if (subDiscountRow) subDiscountRow.style.display = 'flex';
        if (subTierNameEl) subTierNameEl.textContent = `${window.USER_SUBSCRIPTION.tier_name} (${window.USER_SUBSCRIPTION.discount_percent}% OFF)`;
        if (subDiscountEl) subDiscountEl.textContent = `-₹${subscriberDiscount.toFixed(2)}`;
    } else {
        if (subDiscountRow) subDiscountRow.style.display = 'none';
    }

    let totalDiscounts = discountAmount + subscriberDiscount;
    let discountedSubtotal = Math.max(0, subtotal - totalDiscounts);
    const tax = Math.round(discountedSubtotal * 0.05 * 100) / 100;
    const total = Math.round((discountedSubtotal + tax) * 100) / 100;

    const discountRow = document.getElementById('cart-discount-row');
    const discountCodeEl = document.getElementById('coupon-applied-code');
    const discountEl = document.getElementById('cart-discount');

    if (appliedCouponCode && discountAmount > 0) {
        if (discountRow) discountRow.style.display = 'flex';
        if (discountCodeEl) discountCodeEl.textContent = appliedCouponCode;
        if (discountEl) discountEl.textContent = `-₹${discountAmount.toFixed(2)}`;
    } else {
        if (discountRow) discountRow.style.display = 'none';
    }

    if (subtotalEl) subtotalEl.textContent = `₹${subtotal.toFixed(2)}`;
    if (taxEl) taxEl.textContent = `₹${tax.toFixed(2)}`;
    if (totalEl) totalEl.textContent = `₹${total.toFixed(2)}`;
    if (checkoutBtn) checkoutBtn.disabled = false;
}

function addToCart(item) {
    if (!item || !item.id) return;
    const existing = cart.find(i => i.id === item.id);
    if (existing) {
        existing.quantity += 1;
    } else {
        cart.push({ ...item, quantity: 1 });
    }
    saveCart();
    showToast(`Added ${item.name} to cart!`);
    openCart();
}

window.addToCart = addToCart;

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

function initOrderTypeTabs() {
    const cards = document.querySelectorAll('.order-type-card');
    cards.forEach(card => {
        card.addEventListener('click', () => {
            cards.forEach(c => c.classList.remove('active'));
            card.classList.add('active');

            const type = card.dataset.type;
            selectedOrderType = type;

            const panels = document.querySelectorAll('.fulfillment-panel');
            panels.forEach(p => p.classList.add('hidden'));

            const activePanel = document.getElementById(`panel-fulfillment-${type}`);
            if (activePanel) activePanel.classList.remove('hidden');
        });
    });
}

function initPaymentMethodTabs() {
    const cards = document.querySelectorAll('.payment-method-card');
    cards.forEach(card => {
        card.addEventListener('click', () => {
            cards.forEach(c => c.classList.remove('active'));
            card.classList.add('active');

            const method = card.dataset.method;
            selectedPaymentMethod = method;

            const panels = document.querySelectorAll('.payment-details-panel');
            panels.forEach(p => p.classList.remove('active'));

            const activePanel = document.getElementById(`panel-${method}`);
            if (activePanel) activePanel.classList.add('active');
        });
    });
}

function validateFulfillmentFields() {
    if (selectedOrderType === 'Dine-in') {
        const tableInput = document.getElementById('checkout-table-number');
        const tableNumber = tableInput ? tableInput.value.trim() : '';
        if (!tableNumber) {
            showToast('⚠️ Please enter Table Number for Dine-in order');
            if (tableInput) tableInput.focus();
            return false;
        }
    } else if (selectedOrderType === 'Takeaway') {
        const mobileInput = document.getElementById('checkout-takeaway-mobile');
        const mobile = mobileInput ? mobileInput.value.trim() : '';
        if (!mobile) {
            showToast('⚠️ Please enter Mobile Number for Takeaway');
            if (mobileInput) mobileInput.focus();
            return false;
        }
    } else if (selectedOrderType === 'Delivery') {
        const mobileInput = document.getElementById('checkout-delivery-mobile');
        const addressInput = document.getElementById('checkout-delivery-address');
        const mobile = mobileInput ? mobileInput.value.trim() : '';
        const address = addressInput ? addressInput.value.trim() : '';
        if (!mobile) {
            showToast('⚠️ Please enter Mobile Number for Delivery');
            if (mobileInput) mobileInput.focus();
            return false;
        }
        if (!address) {
            showToast('⚠️ Please enter Delivery Address');
            if (addressInput) addressInput.focus();
            return false;
        }
    }
    return true;
}

function initCartDrawer() {
    const openBtn = document.getElementById('open-cart-btn');
    const closeBtn = document.getElementById('close-cart-btn');
    const overlay = document.getElementById('cart-overlay');
    const checkoutBtn = document.getElementById('checkout-btn');

    if (openBtn) openBtn.addEventListener('click', openCart);
    if (closeBtn) closeBtn.addEventListener('click', closeCart);
    if (overlay) overlay.addEventListener('click', closeCart);

    const applyCouponBtn = document.getElementById('apply-coupon-btn');
    if (applyCouponBtn) {
        applyCouponBtn.addEventListener('click', applyCoupon);
    }

    if (checkoutBtn) {
        checkoutBtn.addEventListener('click', async () => {
            if (cart.length === 0) return;

            if (!validateFulfillmentFields()) {
                return;
            }

            // Customer must be logged in before placing order
            if (!window.IS_AUTHENTICATED) {
                openAuthRequiredModal(async () => {
                    if (selectedPaymentMethod === 'Card') {
                        openOtpModal();
                        return;
                    }
                    await executeOrderPlacement(selectedPaymentMethod);
                });
                return;
            }

            if (selectedPaymentMethod === 'Card') {
                openOtpModal();
                return;
            }

            await executeOrderPlacement(selectedPaymentMethod);
        });
    }

    updateCartUI();
}

window.applyUserCoupon = function(code) {
    const input = document.getElementById('coupon-code-input');
    if (input) {
        input.value = code;
        applyCoupon();
    }
};

async function applyCoupon() {
    const input = document.getElementById('coupon-code-input');
    const msgEl = document.getElementById('coupon-message');
    if (!input || !msgEl) return;

    const code = input.value.trim().toUpperCase();
    if (!code) {
        msgEl.style.display = 'block';
        msgEl.style.color = '#ef4444';
        msgEl.textContent = 'Please enter a coupon code.';
        return;
    }

    let subtotal = cart.reduce((sum, i) => sum + (i.price * i.quantity), 0);
    if (subtotal <= 0) {
        msgEl.style.display = 'block';
        msgEl.style.color = '#ef4444';
        msgEl.textContent = 'Add items to cart before applying coupon.';
        return;
    }

    try {
        const resp = await fetch('/api/coupons/validate', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ code: code, subtotal: subtotal })
        });
        const data = await resp.json();
        if (data.valid) {
            appliedCouponCode = data.code;
            discountAmount = data.discount_amount;
            msgEl.style.display = 'block';
            msgEl.style.color = '#10b981';
            msgEl.textContent = `✓ Coupon '${data.code}' applied! Saved ₹${data.discount_amount.toFixed(2)}`;
            updateCartUI();
        } else {
            appliedCouponCode = '';
            discountAmount = 0;
            msgEl.style.display = 'block';
            msgEl.style.color = '#ef4444';
            msgEl.textContent = `✕ ${data.error || 'Invalid coupon code'}`;
            updateCartUI();
        }
    } catch (err) {
        console.error(err);
        msgEl.style.display = 'block';
        msgEl.style.color = '#ef4444';
        msgEl.textContent = 'Error validating coupon.';
    }
}

async function executeOrderPlacement(paymentMethod, txnID) {
    const checkoutBtn = document.getElementById('checkout-btn');
    if (checkoutBtn) {
        checkoutBtn.disabled = true;
        checkoutBtn.innerHTML = '<span>Processing Payment...</span> <i class="fa-solid fa-spinner fa-spin"></i>';
    }

    let tableNumber = '';
    let customerPhone = '';
    let deliveryAddress = '';

    if (selectedOrderType === 'Dine-in') {
        tableNumber = document.getElementById('checkout-table-number')?.value.trim() || 'Table 1';
    } else if (selectedOrderType === 'Takeaway') {
        customerPhone = document.getElementById('checkout-takeaway-mobile')?.value.trim() || '9876543210';
    } else if (selectedOrderType === 'Delivery') {
        customerPhone = document.getElementById('checkout-delivery-mobile')?.value.trim() || '9876543210';
        deliveryAddress = document.getElementById('checkout-delivery-address')?.value.trim() || 'TeaChar Cafe Store';
    }

    const generatedTxnID = txnID || ('TXN' + Math.floor(10000000 + Math.random() * 90000000));
    const paymentStatus = (paymentMethod === 'COD') ? 'Pending' : 'Paid';

    try {
        const response = await fetch('/api/orders', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({
                order_type: selectedOrderType,
                table_number: tableNumber,
                customer_phone: customerPhone,
                delivery_address: deliveryAddress,
                payment_method: paymentMethod,
                payment_status: paymentStatus,
                transaction_id: generatedTxnID,
                coupon_code: appliedCouponCode,
                items: cart.map(item => ({
                    id: item.id,
                    menu_item_id: item.id,
                    name: item.name,
                    item_name: item.name,
                    price: item.price,
                    quantity: item.quantity
                }))
            })
        });

        if (response.ok) {
            cart = [];
            saveCart();
            closeCart();
            showToast(`🎉 Order Paid via ${paymentMethod}! TXN: ${generatedTxnID}`);
            setTimeout(() => {
                window.location.href = '/orders';
            }, 1200);
        } else {
            const errData = await response.json().catch(() => ({}));
            showToast(errData.error || 'Failed to process order. Please check inputs.');
        }
    } catch (err) {
        console.error(err);
        showToast('An error occurred. Please try again.');
    } finally {
        if (checkoutBtn) {
            checkoutBtn.disabled = false;
            checkoutBtn.innerHTML = '<span>Pay & Place Order</span> <i class="fa-solid fa-lock"></i>';
        }
    }
}

function initOtpModal() {
    const modal = document.getElementById('otp-modal');
    const closeBtn = document.getElementById('close-otp-btn');
    const verifyBtn = document.getElementById('verify-otp-btn');

    if (closeBtn) closeBtn.addEventListener('click', closeOtpModal);
    if (verifyBtn) {
        verifyBtn.addEventListener('click', async () => {
            const otpInput = document.getElementById('otp-input');
            if (!otpInput || otpInput.value.length < 4) {
                showToast('Please enter a valid OTP code');
                return;
            }

            closeOtpModal();
            const cardTxnID = 'TXN_CARD_' + Math.floor(100000 + Math.random() * 900000);
            await executeOrderPlacement('Card', cardTxnID);
        });
    }
}

function openOtpModal() {
    const modal = document.getElementById('otp-modal');
    if (modal) modal.classList.add('active');
}

function closeOtpModal() {
    const modal = document.getElementById('otp-modal');
    if (modal) modal.classList.remove('active');
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

window.openCart = openCart;
window.closeCart = closeCart;
window.addToCart = addToCart;

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

window.triggerBuyItem = function(btn) {
    if (!btn) return;
    const id = btn.getAttribute('data-id') || btn.dataset.id;
    const name = btn.getAttribute('data-name') || btn.dataset.name;
    const price = btn.getAttribute('data-price') || btn.dataset.price;
    const image = btn.getAttribute('data-image') || btn.dataset.image;
    if (id && name && price) {
        openQuickOrderModal(id, name, price, image);
    }
};

function initBuyNowButtons() {
    document.addEventListener('click', (e) => {
        const btn = e.target.closest('.buy-now-btn');
        if (btn) {
            e.preventDefault();
            triggerBuyItem(btn);
        }
    });
}

/* ==========================================================================
   Single Item Quick Order Modal Logic
   ========================================================================== */

let currentQuickOrderItem = null;
let currentQuickOrderQty = 1;

window.openQuickOrderModal = function(id, name, price, image) {
    currentQuickOrderItem = { id: parseInt(id), name: name, price: parseFloat(price), image: image };
    currentQuickOrderQty = 1;

    document.getElementById('quick-order-item-id').value = id;
    document.getElementById('quick-order-item-name').value = name;
    document.getElementById('quick-order-item-price').value = price;

    document.getElementById('quick-order-item-title').textContent = name;
    document.getElementById('quick-order-unit-price').textContent = `₹${price}`;
    document.getElementById('quick-order-item-img').src = image || '/static/images/hero-banner.jpg';
    document.getElementById('quick-order-qty').textContent = '1';

    updateQuickOrderTotals();

    const modal = document.getElementById('quick-order-modal');
    if (modal) modal.classList.add('active');
};

window.closeQuickOrderModal = function() {
    const modal = document.getElementById('quick-order-modal');
    if (modal) modal.classList.remove('active');
};

window.changeQuickOrderQty = function(delta) {
    currentQuickOrderQty += delta;
    if (currentQuickOrderQty < 1) currentQuickOrderQty = 1;
    document.getElementById('quick-order-qty').textContent = currentQuickOrderQty;
    updateQuickOrderTotals();
};

function updateQuickOrderTotals() {
    if (!currentQuickOrderItem) return;
    const itemTotal = currentQuickOrderItem.price * currentQuickOrderQty;
    const tax = Math.round(itemTotal * 0.05);
    const grandTotal = itemTotal + tax;
    document.getElementById('quick-order-total-price').textContent = `₹${grandTotal}`;
}

window.updateQuickOrderType = function(radio) {
    document.querySelectorAll('.quick-type-label').forEach(label => {
        label.classList.remove('active');
        label.style.borderColor = 'var(--border-color)';
        label.style.background = '#f8fafc';
        label.style.color = 'var(--text-main)';
    });

    const parentLabel = radio.closest('.quick-type-label');
    if (parentLabel) {
        parentLabel.classList.add('active');
        parentLabel.style.borderColor = 'var(--primary)';
        parentLabel.style.background = 'var(--primary-light)';
        parentLabel.style.color = 'var(--primary)';
    }

    const orderType = radio.value;
    const dineinGroup = document.getElementById('quick-dinein-group');
    const mobileGroup = document.getElementById('quick-mobile-group');
    const deliveryGroup = document.getElementById('quick-delivery-group');

    if (orderType === 'Dine-in') {
        if (dineinGroup) dineinGroup.style.display = 'block';
        if (mobileGroup) mobileGroup.style.display = 'none';
        if (deliveryGroup) deliveryGroup.style.display = 'none';
    } else if (orderType === 'Takeaway') {
        if (dineinGroup) dineinGroup.style.display = 'none';
        if (mobileGroup) mobileGroup.style.display = 'block';
        if (deliveryGroup) deliveryGroup.style.display = 'none';
    } else if (orderType === 'Delivery') {
        if (dineinGroup) dineinGroup.style.display = 'none';
        if (mobileGroup) mobileGroup.style.display = 'block';
        if (deliveryGroup) deliveryGroup.style.display = 'block';
    }
};

window.updateQuickPaymentStyle = function(radio) {
    document.querySelectorAll('.quick-pay-label').forEach(label => {
        label.classList.remove('active');
        label.style.borderColor = 'var(--border-color)';
        label.style.background = '#f8fafc';
        label.style.color = 'var(--text-main)';
    });

    const parentLabel = radio.closest('.quick-pay-label');
    if (parentLabel) {
        parentLabel.classList.add('active');
        parentLabel.style.borderColor = 'var(--primary)';
        parentLabel.style.background = 'var(--primary-light)';
        parentLabel.style.color = 'var(--primary)';
    }
};

window.handleQuickOrderSubmit = async function(event) {
    event.preventDefault();
    if (!currentQuickOrderItem) return;

    const submitBtn = document.getElementById('quick-order-submit-btn');
    const orderType = document.querySelector('input[name="quick_order_type"]:checked')?.value || 'Dine-in';
    const paymentMethod = document.querySelector('input[name="quick_payment"]:checked')?.value || 'UPI';
    const note = document.getElementById('quick-order-note')?.value.trim() || '';

    let tableNumber = '';
    let phone = '';
    let address = '';

    if (orderType === 'Dine-in') {
        tableNumber = document.getElementById('quick-order-table').value.trim();
        if (!tableNumber) {
            alert('Please enter your table number for Dine-in orders.');
            return;
        }
    } else if (orderType === 'Takeaway') {
        phone = document.getElementById('quick-order-phone').value.trim();
        if (!phone) {
            alert('Please enter your mobile number for Takeaway orders.');
            return;
        }
    } else if (orderType === 'Delivery') {
        phone = document.getElementById('quick-order-phone').value.trim();
        address = document.getElementById('quick-order-address').value.trim();
        if (!phone) {
            alert('Please enter your mobile number for Delivery orders.');
            return;
        }
        if (!address) {
            alert('Please enter your delivery address.');
            return;
        }
    }

    // Check if customer is authenticated before placing quick order
    if (!window.IS_AUTHENTICATED) {
        openAuthRequiredModal(async () => {
            await doQuickOrderPlacement(submitBtn, orderType, paymentMethod, tableNumber, phone, address, note);
        });
        return;
    }

    await doQuickOrderPlacement(submitBtn, orderType, paymentMethod, tableNumber, phone, address, note);
};

async function doQuickOrderPlacement(submitBtn, orderType, paymentMethod, tableNumber, phone, address, note) {
    if (submitBtn) {
        submitBtn.disabled = true;
        submitBtn.innerHTML = '<i class="fa-solid fa-spinner fa-spin me-1"></i> Placing Order...';
    }

    const itemTotal = currentQuickOrderItem.price * currentQuickOrderQty;
    const tax = Math.round(itemTotal * 0.05);

    const payload = {
        order_type: orderType,
        table_number: tableNumber,
        customer_phone: phone,
        delivery_address: address,
        address: address,
        items: [{
            id: currentQuickOrderItem.id,
            menu_item_id: currentQuickOrderItem.id,
            name: currentQuickOrderItem.name,
            item_name: currentQuickOrderItem.name,
            quantity: currentQuickOrderQty,
            price: currentQuickOrderItem.price
        }],
        subtotal: itemTotal,
        subtotal_price: itemTotal,
        total: itemTotal + tax,
        total_price: itemTotal + tax,
        payment_method: paymentMethod
    };

    try {
        const response = await fetch('/api/orders', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(payload)
        });

        if (response.ok) {
            const result = await response.json();
            closeQuickOrderModal();
            showToast('Order placed successfully!');
            setTimeout(() => {
                window.location.href = '/orders';
            }, 800);
        } else {
            const errorText = await response.text();
            alert(`Order placement error: ${errorText}`);
            if (submitBtn) {
                submitBtn.disabled = false;
                submitBtn.innerHTML = '<i class="fa-solid fa-circle-check me-1"></i> Place Order';
            }
        }
    } catch (err) {
        alert(`Network error: ${err.message}`);
        if (submitBtn) {
            submitBtn.disabled = false;
            submitBtn.innerHTML = '<i class="fa-solid fa-circle-check me-1"></i> Place Order';
        }
    }
}

// Global Theme-Aligned Confirmation Modal Handlers
let pendingConfirmCallback = null;

window.showConfirmModal = function(title, message, confirmText, callback) {
    const modal = document.getElementById('theme-confirm-modal');
    if (!modal) {
        if (confirm(message)) callback();
        return;
    }

    const titleEl = document.getElementById('theme-confirm-title');
    if (titleEl) titleEl.innerHTML = `<i class="fa-solid fa-circle-exclamation text-warning me-1"></i> ${title || 'Confirm Action'}`;

    const msgEl = document.getElementById('theme-confirm-message');
    if (msgEl) msgEl.textContent = message || 'Are you sure you want to proceed?';
    
    const confirmBtn = document.getElementById('theme-confirm-btn');
    if (confirmBtn) {
        confirmBtn.innerHTML = `<i class="fa-solid fa-check me-1"></i> ${confirmText || 'Confirm'}`;
        confirmBtn.onclick = function() {
            closeThemeConfirmModal();
            if (callback) callback();
        };
    }

    pendingConfirmCallback = callback;
    modal.classList.add('active');
};

window.closeThemeConfirmModal = function() {
    const modal = document.getElementById('theme-confirm-modal');
    if (modal) modal.classList.remove('active');
    pendingConfirmCallback = null;
};

window.confirmFormAction = function(formElem, title, message, confirmText) {
    window.showConfirmModal(title, message, confirmText, function() {
        formElem.submit();
    });
    return false;
};

/* ==========================================================================
   Customer Login Required Modal Management
   ========================================================================== */

window.openAuthRequiredModal = function(onSuccessCallback) {
    pendingOrderAction = onSuccessCallback;
    const modal = document.getElementById('auth-required-modal');
    const alertEl = document.getElementById('modal-login-alert');
    if (alertEl) alertEl.style.display = 'none';

    if (modal) modal.classList.add('active');
    const emailInput = document.getElementById('modal-login-email');
    if (emailInput) emailInput.focus();
};

window.closeAuthRequiredModal = function() {
    const modal = document.getElementById('auth-required-modal');
    if (modal) modal.classList.remove('active');
    pendingOrderAction = null;
};

window.fillModalCustomerDemo = function() {
    const email = document.getElementById('modal-login-email');
    const pass = document.getElementById('modal-login-password');
    if (email) email.value = 'client@teachar.in';
    if (pass) pass.value = 'Client@123';
};

window.handleModalLoginSubmit = async function(event) {
    event.preventDefault();
    const email = document.getElementById('modal-login-email')?.value.trim();
    const password = document.getElementById('modal-login-password')?.value;
    const alertEl = document.getElementById('modal-login-alert');
    const errorEl = document.getElementById('modal-login-error');
    const submitBtn = document.getElementById('modal-login-btn');

    if (!email || !password) return;

    if (submitBtn) {
        submitBtn.disabled = true;
        submitBtn.innerHTML = '<i class="fa-solid fa-spinner fa-spin me-1"></i> Authenticating...';
    }

    try {
        const response = await fetch('/api/auth/login', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ email, password })
        });

        const data = await response.json().catch(() => ({}));

        if (response.ok && data.success) {
            window.IS_AUTHENTICATED = true;
            window.CURRENT_USER = data.user;

            showToast(`Welcome back, ${data.user.name || 'Customer'}!`);
            
            const modal = document.getElementById('auth-required-modal');
            if (modal) modal.classList.remove('active');

            // Re-populate phone and address if present
            const phoneInput = document.getElementById('checkout-takeaway-mobile') || document.getElementById('checkout-delivery-mobile') || document.getElementById('quick-order-phone');
            if (phoneInput && data.user.mobile_number && !phoneInput.value) {
                phoneInput.value = data.user.mobile_number;
            }

            const addressInput = document.getElementById('checkout-delivery-address') || document.getElementById('quick-order-address');
            if (addressInput && data.user.address && !addressInput.value) {
                addressInput.value = data.user.address;
            }

            // Immediately continue the pending order placement action
            if (typeof pendingOrderAction === 'function') {
                const action = pendingOrderAction;
                pendingOrderAction = null;
                await action();
            }
        } else {
            if (alertEl && errorEl) {
                errorEl.textContent = data.error || 'Invalid email or password. Please try again.';
                alertEl.style.display = 'block';
            }
        }
    } catch (err) {
        if (alertEl && errorEl) {
            errorEl.textContent = 'Connection error. Please try again.';
            alertEl.style.display = 'block';
        }
    } finally {
        if (submitBtn) {
            submitBtn.disabled = false;
            submitBtn.innerHTML = '<span>Sign In & Continue Order</span> <i class="fa-solid fa-arrow-right"></i>';
        }
    }
};

/* ==========================================================================
   Add To Favorite Feature
   ========================================================================== */

function initFavorites() {
    updateFavoriteButtonsUI();
    updateFavoritesBadgeCount();
}

function updateFavoritesBadgeCount() {
    const badge = document.getElementById('favorites-count-badge');
    if (badge) {
        badge.textContent = favorites.length;
    }
}

function isFavorite(id) {
    const numId = parseInt(id, 10);
    return favorites.some(item => (item.id === numId || item === numId));
}

window.toggleFavorite = function(item, btn) {
    if (!item || !item.id) return;
    const numId = parseInt(item.id, 10);
    const existingIndex = favorites.findIndex(f => (typeof f === 'object' ? f.id === numId : f === numId));

    if (existingIndex > -1) {
        favorites.splice(existingIndex, 1);
        showToast(`Removed "${item.name}" from favorites`);
    } else {
        favorites.push({
            id: numId,
            name: item.name,
            price: item.price,
            image: item.image
        });
        showToast(`❤️ Added "${item.name}" to favorites!`);
    }

    localStorage.setItem('teachar_favorites', JSON.stringify(favorites));
    updateFavoriteButtonsUI();
    updateFavoritesBadgeCount();

    // If favorites tab is active, re-filter view
    const activeTab = document.querySelector('.tab-btn.active');
    if (activeTab && activeTab.dataset.category === 'favorites') {
        filterMenu('favorites', getSearchTerm());
    }
};

// Global click event listener for .fav-btn ensuring reliable clicks
document.addEventListener('click', (e) => {
    const btn = e.target.closest('.fav-btn');
    if (btn) {
        e.preventDefault();
        e.stopPropagation();
        const id = parseInt(btn.dataset.id, 10);
        const name = btn.dataset.name || 'Item';
        const price = parseFloat(btn.dataset.price) || 0;
        const image = btn.dataset.image || '';
        toggleFavorite({ id, name, price, image }, btn);
    }
});

function updateFavoriteButtonsUI() {
    document.querySelectorAll('.fav-btn').forEach(btn => {
        const id = parseInt(btn.dataset.id, 10);
        if (isFavorite(id)) {
            btn.classList.add('active');
            btn.innerHTML = '<i class="fa-solid fa-heart"></i>';
            btn.title = 'Remove from Favorites';
        } else {
            btn.classList.remove('active');
            btn.innerHTML = '<i class="fa-regular fa-heart"></i>';
            btn.title = 'Add to Favorites';
        }
    });
}

// Override filterMenu to support 'favorites' filter
const originalFilterMenu = window.filterMenu || filterMenu;
window.filterMenu = function(category, searchTerm) {
    const categoryBlocks = document.querySelectorAll('.menu-category-block');
    const cards = document.querySelectorAll('.menu-card');
    const noResultsEl = document.getElementById('no-search-results');

    let visibleCardsCount = 0;

    cards.forEach(card => {
        const itemId = parseInt(card.dataset.itemId || card.getAttribute('data-item-id'), 10);
        const itemCategory = card.dataset.category || card.getAttribute('data-category');
        const itemName = (card.dataset.name || card.getAttribute('data-name') || '').toLowerCase();

        let matchesCategory = false;
        if (category === 'all') {
            matchesCategory = true;
        } else if (category === 'favorites') {
            matchesCategory = isFavorite(itemId);
        } else {
            matchesCategory = (itemCategory === category);
        }

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
            if (category === 'favorites') {
                noResultsEl.innerHTML = `
                    <i class="fa-regular fa-heart" style="font-size:2.5rem; color:#ef4444; margin-bottom:0.75rem;"></i>
                    <h3 style="margin-bottom:0.4rem;">No Favorites Yet</h3>
                    <p style="color:var(--text-muted); font-size:0.9rem;">Click the heart icon on any tea, coffee, or snack to save your favorites here!</p>
                `;
            } else {
                noResultsEl.innerHTML = `
                    <i class="fa-solid fa-circle-question" style="font-size:2.5rem; color:var(--text-muted); margin-bottom:0.75rem;"></i>
                    <h3 style="margin-bottom:0.4rem;">No matching items found</h3>
                    <p style="color:var(--text-muted); font-size:0.9rem;">Try adjusting your search or category filter</p>
                `;
            }
            noResultsEl.classList.remove('hidden');
        } else {
            noResultsEl.classList.add('hidden');
        }
    }
};