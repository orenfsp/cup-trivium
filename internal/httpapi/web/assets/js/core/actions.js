// Делегированный диспетчер действий. Вместо инлайн-onclick и глобальных
// функций разметка помечает элементы атрибутом data-action="имя" (плюс
// опционально data-arg и другие data-*), а модули регистрируют обработчики.
// Это делает безопасными динамически отрендеренные списки: идентификаторы
// передаются как данные, а не как JS-строки в атрибуте.
import { toast } from './dom.js';

const handlers = new Map();

/** Зарегистрировать набор действий: { 'имя-действия': async (arg, el) => {} }. */
export function registerActions(map) {
  for (const [name, fn] of Object.entries(map)) handlers.set(name, fn);
}

/** Единожды подключить делегирование кликов на корневом элементе. */
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
