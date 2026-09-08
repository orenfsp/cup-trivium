// Категории обращений: общий справочник для двух профилей — селекта анкеты
// заявителя и селекта в карточке сотрудника (лишний на конкретном порту
// просто скрыт, поэтому заполняются оба).
// ТЗ п.3: «Не знаю, как это назвать» (free_form) — обязательный пункт
// списка заявителя: он ведёт на свободное описание, бэкенд сам переводит
// обращение в режим free_text.
import { $, esc } from './core/dom.js';
import { api } from './core/api.js';

export let categories = [];

export async function loadCategories() {
  try {
    const cats = (await api('GET', '/api/categories')).categories || [];
    categories = cats;
    // Список заявителя: обычные категории + free_form-пункт последним.
    const plain = cats.filter((c) => !c.free_form);
    const free = cats.filter((c) => c.free_form);
    $('apCat').innerHTML = plain.map((c) => `<option value="${c.id}">${esc(c.name)}</option>`).join('') +
      free.map((c) => `<option value="${c.id}">${esc(c.name)} — своими словами</option>`).join('');
    $('dCategory').innerHTML = categories.map((c) => `<option value="${c.id}">${esc(c.name)}</option>`).join('');
  } catch (e) {
    console.error(e);
  }
}
