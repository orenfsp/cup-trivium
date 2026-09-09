import { $, esc } from './core/dom.js';
import { api } from './core/api.js';

export let categories = [];

export async function loadCategories() {
  try {
    const cats = (await api('GET', '/api/categories')).categories || [];
    categories = cats;
    const plain = cats.filter((c) => !c.free_form);
    const free = cats.filter((c) => c.free_form);
    const apCat = $('apCat'); // селект есть только на странице подачи обращения
    if (apCat) {
      apCat.innerHTML = plain.map((c) => `<option value="${c.id}">${esc(c.name)}</option>`).join('') +
        free.map((c) => `<option value="${c.id}">${esc(c.name)} — своими словами</option>`).join('');
    }
    const dCat = $('dCategory'); // селект есть только в карточке обращения сотрудника
    if (dCat) {
      dCat.innerHTML = categories.map((c) => `<option value="${c.id}">${esc(c.name)}</option>`).join('');
    }
  } catch (e) {
    console.error(e);
  }
}
