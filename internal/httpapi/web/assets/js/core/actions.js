import { toast } from './dom.js';

const handlers = new Map();

export function registerActions(map) {
  for (const [name, fn] of Object.entries(map)) handlers.set(name, fn);
}

export function bindActionDelegation(root = document) {
  root.addEventListener('click', async (e) => {
    const el = e.target.closest('[data-action]');
    if (!el) return;
    const fn = handlers.get(el.dataset.action);
    if (!fn) return;
    e.preventDefault();
    try {
      await fn(el.dataset.arg, el);
    } catch (err) {
      toast(err?.message || String(err)); // страховочная сеть для необработанных ошибок
    }
  });
}
