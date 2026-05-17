'use strict';

// ─────────────────────────────────────────────────────────────────
//  POS Offline Manager — IndexedDB cart persistence
// ─────────────────────────────────────────────────────────────────

const POS_DB_NAME = 'omnichannel_pos';
const POS_DB_VERSION = 1;
const CART_STORE_NAME = 'cart_items';

class POSOfflineManager {
  constructor() {
    this.db = null;
    this.isOnline = navigator.onLine;
    this._init();
    this._setupNetworkListeners();
  }

  async _init() {
    try {
      this.db = await this._openDB();
      this._updateOfflineBadge();
    } catch (err) {
      console.warn('[POS] IndexedDB init failed:', err);
    }
  }

  _openDB() {
    return new Promise((resolve, reject) => {
      const req = indexedDB.open(POS_DB_NAME, POS_DB_VERSION);
      req.onupgradeneeded = (e) => {
        const db = e.target.result;
        if (!db.objectStoreNames.contains(CART_STORE_NAME)) {
          const store = db.createObjectStore(CART_STORE_NAME, { keyPath: 'product_id' });
          store.createIndex('updated_at', 'updated_at', { unique: false });
        }
      };
      req.onsuccess  = (e) => resolve(e.target.result);
      req.onerror    = (e) => reject(e.target.error);
    });
  }

  _setupNetworkListeners() {
    window.addEventListener('online', () => {
      this.isOnline = true;
      this._updateOfflineBadge();
      this._syncPendingCart();
    });
    window.addEventListener('offline', () => {
      this.isOnline = false;
      this._updateOfflineBadge();
    });

    // Intercept HTMX requests when offline — queue them locally
    document.addEventListener('htmx:beforeRequest', (e) => {
      if (!this.isOnline) {
        const path = e.detail?.requestConfig?.path ?? '';
        if (path.includes('/cart/')) {
          e.preventDefault();
          this._handleOfflineCartOp(e.detail);
        }
      }
    });

    // After HTMX swaps cart items, sync subtotal to Alpine state
    document.addEventListener('htmx:afterSwap', (e) => {
      if (e.target?.id === 'cart-items') {
        const subtotalInput = document.getElementById('cart-subtotal-val');
        const countInput    = document.getElementById('cart-item-count');
        const alpine = document.querySelector('[x-data]');
        if (alpine && subtotalInput) {
          const alpineComp = alpine._x_dataStack?.[0];
          if (alpineComp) {
            alpineComp.subtotal  = parseInt(subtotalInput.value, 10) || 0;
            alpineComp.itemCount = parseInt(countInput?.value ?? '0', 10) || 0;
          }
        }
      }
    });
  }

  _updateOfflineBadge() {
    const badge = document.getElementById('offline-badge');
    if (badge) {
      badge.style.display = this.isOnline ? 'none' : 'flex';
    }
  }

  async saveCartItem(item) {
    if (!this.db) return;
    const tx    = this.db.transaction(CART_STORE_NAME, 'readwrite');
    const store = tx.objectStore(CART_STORE_NAME);
    const existing = await new Promise(r => { const q = store.get(item.product_id); q.onsuccess = () => r(q.result); });
    if (existing) {
      item.quantity = (existing.quantity || 0) + (item.quantity || 1);
    }
    item.updated_at = Date.now();
    store.put(item);
  }

  async getCartItems() {
    if (!this.db) return [];
    const tx    = this.db.transaction(CART_STORE_NAME, 'readonly');
    const store = tx.objectStore(CART_STORE_NAME);
    return new Promise((resolve, reject) => {
      const req = store.getAll();
      req.onsuccess = () => resolve(req.result);
      req.onerror   = () => reject(req.error);
    });
  }

  async clearCart() {
    if (!this.db) return;
    const tx = this.db.transaction(CART_STORE_NAME, 'readwrite');
    tx.objectStore(CART_STORE_NAME).clear();
  }

  _handleOfflineCartOp(detail) {
    const path   = detail?.requestConfig?.path ?? '';
    const params = detail?.requestConfig?.parameters ?? {};
    if (path.includes('/cart/add') && params.product_id) {
      this.saveCartItem({
        product_id: params.product_id,
        name:       params.name  ?? 'Unknown',
        price:      parseInt(params.price ?? '0', 10),
        quantity:   1,
      });
    }
  }

  async _syncPendingCart() {
    const items = await this.getCartItems();
    if (items.length === 0) return;
    try {
      await fetch('/api/pos/cart/sync', {
        method:  'POST',
        headers: { 'Content-Type': 'application/json' },
        body:    JSON.stringify({ items }),
      });
    } catch (err) {
      console.warn('[POS] Sync failed:', err);
    }
  }
}

const posOffline = new POSOfflineManager();

// ─────────────────────────────────────────────────────────────────
//  Alpine.js cartManager() — reactive cart state
// ─────────────────────────────────────────────────────────────────

function cartManager() {
  return {
    // State
    itemCount:         0,
    subtotal:          0,
    discountAmount:    0,
    showPaymentModal:  false,
    paymentMethod:     'tunai',
    cashReceived:      0,
    processingPayment: false,

    // Quick cash amount options: exact + round-ups
    get quickAmounts() {
      return [0, 1000, 5000, 10000];
    },

    // Computed
    get grandTotal() {
      return Math.max(0, this.subtotal - this.discountAmount);
    },
    get kembalian() {
      return this.cashReceived - this.grandTotal;
    },

    // Methods
    openPaymentModal() {
      if (this.itemCount === 0) return;
      this.cashReceived = this.grandTotal;
      this.showPaymentModal = true;
    },
    closePaymentModal() {
      this.showPaymentModal = false;
    },
    clearCart() {
      if (!confirm('Kosongkan semua item dari keranjang?')) return;
      fetch('/api/pos/cart/clear', { method: 'DELETE' })
        .then(() => {
          document.getElementById('cart-items').innerHTML = '';
          this.subtotal    = 0;
          this.itemCount   = 0;
          posOffline.clearCart();
        })
        .catch(console.warn);
    },
    async processPayment() {
      if (this.paymentMethod === 'tunai' && this.kembalian < 0) return;
      this.processingPayment = true;
      try {
        const resp = await fetch('/api/pos/orders', {
          method:  'POST',
          headers: { 'Content-Type': 'application/json' },
          body:    JSON.stringify({
            payment_method: this.paymentMethod,
            discount:       this.discountAmount,
            cash_received:  this.paymentMethod === 'tunai' ? this.cashReceived : 0,
          }),
        });
        if (resp.ok) {
          this.showPaymentModal = false;
          this.subtotal         = 0;
          this.itemCount        = 0;
          this.discountAmount   = 0;
          this.cashReceived     = 0;
          document.getElementById('cart-items').innerHTML = '';
          await posOffline.clearCart();
          // Flash success
          this._showFlash('Pembayaran berhasil!', 'success');
        } else {
          this._showFlash('Pembayaran gagal. Coba lagi.', 'error');
        }
      } catch (err) {
        this._showFlash('Gagal terhubung ke server.', 'error');
      } finally {
        this.processingPayment = false;
      }
    },
    currentTime() {
      return new Date().toLocaleTimeString('id-ID', { hour: '2-digit', minute: '2-digit', second: '2-digit' });
    },
    formatRupiah(amount) {
      if (!amount || amount <= 0) return 'Rp 0';
      return new Intl.NumberFormat('id-ID', {
        style:                 'currency',
        currency:              'IDR',
        minimumFractionDigits: 0,
        maximumFractionDigits: 0,
      }).format(amount);
    },
    formatRupiahShort(amount) {
      if (amount >= 1_000_000) return (amount / 1_000_000).toFixed(1) + 'jt';
      if (amount >= 1_000)     return (amount / 1_000).toFixed(0) + 'rb';
      return String(amount);
    },
    _showFlash(msg, type) {
      const el = document.createElement('div');
      el.className = `fixed top-4 right-4 z-[60] px-4 py-3 rounded-xl shadow-lg text-sm font-medium ${
        type === 'success' ? 'bg-green-600 text-white' : 'bg-red-600 text-white'
      }`;
      el.textContent = msg;
      document.body.appendChild(el);
      setTimeout(() => el.remove(), 3000);
    },
  };
}

// Live clock update every second
setInterval(() => {
  const el = document.querySelector('[x-text="currentTime()"]');
  if (el && window.Alpine) {
    // Alpine handles reactivity; just trigger a no-op to force re-evaluation
  }
}, 1000);
