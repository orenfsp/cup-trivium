const STAFF_TOKEN_KEY = 'otklik_staff_token';

export const staffToken = () => sessionStorage.getItem(STAFF_TOKEN_KEY);
export const setStaffToken = (t) => sessionStorage.setItem(STAFF_TOKEN_KEY, t);
export const clearStaffToken = () => sessionStorage.removeItem(STAFF_TOKEN_KEY);

export async function api(method, path, body) {
  const opt = { method, headers: {} };
  const tk = staffToken();
  if (tk) opt.headers['Authorization'] = 'Bearer ' + tk;
  if (body !== undefined) {
    opt.headers['Content-Type'] = 'application/json; charset=utf-8';
    opt.body = JSON.stringify(body);
  }
  const res = await fetch(path, opt);
  let data = null;
  try { data = await res.json(); } catch (e) { /* пустое тело — это нормально */ }
  if (!res.ok) throw new Error(data && data.error ? data.error : 'HTTP ' + res.status);
  return data;
}
