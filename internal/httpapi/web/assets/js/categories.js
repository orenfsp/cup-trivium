import { $, esc } from './core/dom.js';
import { api } from './core/api.js';

export let categories = [];

export async function loadCategories() {
  try {
    const cats = (await api('GET', '/api/categories')).categories || [];
    categories = cats;
    const plain = cats.filter((c) => !c.free_form);
    const free = cats.filter((c) => c.free_form);
    $('apCat').innerHTML = plain.map((c) => `<option value="${c.id}">${esc(c.name)}</option>`).join('') +
      free.map((c) => `<option value="${c.id}">${esc(c.name)} — своими словами</option>`).join('');
    $('dCategory').innerHTML = categories.map((c) => `<option value="${c.id}">${esc(c.name)}</option>`).join('');
  } catch (e) {
    console.error(e);
  }
}
