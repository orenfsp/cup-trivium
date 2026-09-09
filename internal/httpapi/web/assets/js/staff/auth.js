import { $, showErr, hideErr, toast } from '../core/dom.js';
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

// Смена пароля самим сотрудником: /api/auth/password инвалидирует прочие сессии.
export async function changePassword() {
  const current = window.prompt('Текущий пароль:');
  if (!current) return;
  const next = window.prompt('Новый пароль (минимум 8 символов):');
  if (!next) return;
  if (next.length < 8) { toast('Пароль должен быть не короче 8 символов'); return; }
  const confirm2 = window.prompt('Повторите новый пароль:');
  if (confirm2 !== next) { toast('Пароли не совпадают'); return; }
  try {
    await api('POST', '/api/auth/password', { current_password: current, new_password: next });
    toast('Пароль изменён; другие сессии завершены');
  } catch (e) {
    toast('Ошибка: ' + e.message);
  }
}

registerActions({
  'staff-login': login,
  'staff-logout': logout,
  'staff-change-password': changePassword,
});
