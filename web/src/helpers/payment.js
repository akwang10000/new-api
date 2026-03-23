export function isQRCodePaymentLink(payLink) {
  if (!payLink) {
    return false;
  }
  const normalized = String(payLink).trim();
  if (
    normalized.startsWith('weixin://') ||
    normalized.startsWith('alipays://')
  ) {
    return true;
  }
  try {
    const parsed = new URL(normalized, window.location.origin);
    const host = parsed.hostname.toLowerCase();
    return host === 'qr.alipay.com';
  } catch {
    return false;
  }
}

const PAYMENT_CHECKOUT_STORAGE_PREFIX = 'payment_checkout:';
const PAYMENT_CHECKOUT_TTL_MS = 30 * 60 * 1000;

function cleanupExpiredPaymentCheckouts() {
  try {
    const now = Date.now();
    const keys = [];
    for (let index = 0; index < window.localStorage.length; index += 1) {
      const key = window.localStorage.key(index);
      if (key) {
        keys.push(key);
      }
    }
    keys.forEach((key) => {
      if (!key.startsWith(PAYMENT_CHECKOUT_STORAGE_PREFIX)) {
        return;
      }
      const raw = window.localStorage.getItem(key);
      if (!raw) {
        window.localStorage.removeItem(key);
        return;
      }
      try {
        const parsed = JSON.parse(raw);
        if (!parsed?.expires_at || parsed.expires_at <= now) {
          window.localStorage.removeItem(key);
        }
      } catch {
        window.localStorage.removeItem(key);
      }
    });
  } catch {
    // Ignore storage failures and fall back to opening the payment link directly.
  }
}

function createPaymentCheckoutId() {
  if (window.crypto?.randomUUID) {
    return window.crypto.randomUUID();
  }
  return `${Date.now()}-${Math.random().toString(36).slice(2, 10)}`;
}

function persistPaymentCheckout(checkout) {
  cleanupExpiredPaymentCheckouts();
  const checkoutId = createPaymentCheckoutId();
  window.localStorage.setItem(
    `${PAYMENT_CHECKOUT_STORAGE_PREFIX}${checkoutId}`,
    JSON.stringify({
      checkout,
      expires_at: Date.now() + PAYMENT_CHECKOUT_TTL_MS,
    }),
  );
  return checkoutId;
}

export function loadPaymentCheckout(checkoutId) {
  if (!checkoutId) {
    return null;
  }
  cleanupExpiredPaymentCheckouts();
  try {
    const raw = window.localStorage.getItem(
      `${PAYMENT_CHECKOUT_STORAGE_PREFIX}${checkoutId}`,
    );
    if (!raw) {
      return null;
    }
    const payload = JSON.parse(raw);
    if (!payload?.expires_at || payload.expires_at <= Date.now()) {
      window.localStorage.removeItem(
        `${PAYMENT_CHECKOUT_STORAGE_PREFIX}${checkoutId}`,
      );
      return null;
    }
    return payload.checkout || null;
  } catch {
    return null;
  }
}

export function openPaymentCheckout(checkout = {}) {
  const payLink = String(checkout?.pay_link || '').trim();
  if (!payLink) {
    return false;
  }

  const qrContent = String(checkout?.qr_content || '').trim();
  const payLinkType = String(checkout?.pay_link_type || '').trim();
  const shouldRenderQRCode =
    (payLinkType === 'qrcode' && qrContent) || isQRCodePaymentLink(payLink);

  if (shouldRenderQRCode) {
    let checkoutId = '';
    try {
      checkoutId = persistPaymentCheckout({
        ...checkout,
        pay_link: payLink,
        qr_content: qrContent || payLink,
      });
    } catch {
      checkoutId = '';
    }
    if (!checkoutId) {
      return false;
    }
    const pageURL = new URL('/payment/qrcode', window.location.origin);
    pageURL.searchParams.set('checkout_id', checkoutId);
    window.open(pageURL.toString(), '_blank', 'noopener,noreferrer');
    return true;
  }

  window.open(payLink, '_blank', 'noopener,noreferrer');
  return true;
}
