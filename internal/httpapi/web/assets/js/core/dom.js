export const $ = (id) => document.getElementById(id);

export const esc = (s) => String(s ?? '').replace(/[&<>"]/g,
  (c) => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;' }[c]));

export const shortId = (s) => (s ? String(s).slice(0, 8) : '—');

export const fmtTime = (s) => (s ? new Date(s).toLocaleString('ru-RU') : '—');

export const badge = (cls, text) => `<span class="badge ${esc(cls)}">${esc(text)}</span>`;

export const kv = (k, v) =>
  `<div class="kv"><b>${esc(k)}:</b><span>${v == null || v === '' ? '—' : v}</span></div>`;

export function toast(msg) {
  const t = document.createElement('div');
  t.textContent = msg;
  t.style.cssText = 'position:fixed;left:50%;transform:translateX(-50%);bottom:calc(20px + env(safe-area-inset-bottom,0px));background:#111827;color:#fff;padding:12px 20px;border-radius:12px;z-index:99;font-size:14.5px;max-width:88vw;text-align:center';
  document.body.appendChild(t);
  setTimeout(() => t.remove(), 3200);
}

export function showErr(id, err) {
  const el = $(id);
  el.textContent = err.message || String(err);
  el.classList.remove('hidden');
}

export function hideErr(id) {
  $(id).classList.add('hidden');
}
