import { $ } from '../core/dom.js';
import { api } from '../core/api.js';
import { ruRole } from '../core/i18n.js';
import { loadCategories } from '../categories.js';
import { initStaffAuth } from './auth.js';
import { loadQueue, loadOpAll, loadExpert, bindListFilters, loadMyStats, loadOpComplaints } from './lists.js';
import { closeDetail } from './detail.js';
import { loadAdminAppeals, loadAdminUsers, loadAdminCats, loadStats, loadAdminSettings } from './admin.js';
import { staffState } from './state.js';

function afterStaffLogin() {
  const me = staffState.me;
  $('loginCard').classList.add('hidden');
  $('stHome').classList.remove('hidden');
  $('whoami').textContent = `${me.login} (${ruRole(me.role)})`;
  $('stHello').textContent = `${me.login} — ${ruRole(me.role)}`;
  $('opPanel').classList.toggle('hidden', !(me.role === 'operator' || me.role === 'admin'));
  $('exPanel').classList.toggle('hidden', me.role !== 'expert');
  $('adPanel').classList.toggle('hidden', me.role !== 'admin');
  if (me.role === 'operator' || me.role === 'admin') { loadQueue(); loadOpAll(); loadOpComplaints(); }
  if (me.role === 'operator' || me.role === 'expert') loadMyStats();
  if (me.role === 'expert') loadExpert();
  if (me.role === 'admin') { loadAdminAppeals(); loadAdminUsers(); loadAdminCats(); loadStats(); loadAdminSettings(); }
}

function showLoginScreen() {
  $('stHome').classList.add('hidden');
  closeDetail(); // заодно останавливает поллинг карточки
  $('loginCard').classList.remove('hidden');
  $('whoami').textContent = 'анонимно';
}

export async function staffInit() {
  initStaffAuth({ afterLogin: afterStaffLogin, afterLogout: showLoginScreen });
  bindListFilters();
  await loadCategories(); // селект категорий в карточке обращения
  try {
    const r = await api('GET', '/api/me');
    if (r.user_id) {
      staffState.me = r;
      afterStaffLogin();
    }
  } catch (e) { /* нет сессии — покажем форму входа */ }
}
