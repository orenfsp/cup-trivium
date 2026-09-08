// Вход/выход сотрудника. Сессия — токен вкладки (sessionStorage) либо кука;
// за дальнейшую сборку панелей отвечает оркестратор (staff/index.js),
// поэтому модуль получает хуки afterLogin/afterLogout через init.
import { $, showErr, hideErr } from '../core/dom.js';
import { api, setStaffToken, clearStaffToken } from '../core/api.js';
import { registerActions } from '../core/actions.js';
import { staffState } from './state.js';

let hooks = { afterLogin: null, afterLogout: null };

export function initStaffAuth(h) {
  hooks = h;
}

export async function login() {
  hideErr('stErr');
  try {
    const me = await api('POST', '/api/auth/login', {
      login: $('stLogin').value.trim(),
      password: $('stPass').value
    });
    // Токен сохраняем в sessionStorage — он уникален для каждой вкладки,
    // поэтому в соседних вкладках можно параллельно работать под другими
    // сотрудниками.
    if (me.session_token) setStaffToken(me.session_token);
    staffState.me = me;
    if (hooks.afterLogin) await hooks.afterLogin();
  } catch (e) {
    showErr('stErr', e);
  }
}

export async function logout() {
  try { await api('POST', '/api/auth/logout'); } catch (e) { /* сессия могла истечь */ }
  staffState.me = null;
  clearStaffToken();
  if (hooks.afterLogout) await hooks.afterLogout();
}

registerActions({
  'staff-login': login,
  'staff-logout': logout,
});
