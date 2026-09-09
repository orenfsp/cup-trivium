import { $ } from '../core/dom.js';
import { api } from '../core/api.js';
import { intakeInit } from './intake.js';
import { saveTrack, renderSaved, verifyTrack } from './track.js';
import { loadApplicantView, viewPoller } from './view.js';

// Возвращает сессию заявителя или null; заодно подписывает шапку.
async function applicantSession() {
  try {
    const r = await api('GET', '/api/me');
    if (r.role === 'applicant') {
      $('whoami').textContent = 'заявитель';
      return r;
    }
  } catch (e) { /* нет активной сессии заявителя — нормальная ситуация */ }
  return null;
}

// Страница /new — форма подачи обращения.
export async function newInit() {
  await intakeInit();
}

// Страница /track — вход по трек-номеру.
export async function trackInit() {
  renderSaved();
  const urlTrack = new URLSearchParams(location.search).get('track');
  if (urlTrack) {
    $('trkInput').value = urlTrack;
    history.replaceState(null, '', location.pathname); // убираем номер из адресной строки
    verifyTrack(); // при успехе уведёт на /appeal
  }
}

// Страница /appeal — статус и чат по обращению заявителя.
export async function appealInit() {
  // Прямая ссылка из QR-кода или файла-памятки: /appeal?track=ОТК-...
  const urlTrack = new URLSearchParams(location.search).get('track');
  if (urlTrack) {
    saveTrack(urlTrack);
    history.replaceState(null, '', location.pathname);
    try {
      await api('POST', '/api/appeals/track', { track_number: urlTrack });
    } catch (e) { /* неверный номер — ниже отправим на /track */ }
  }
  const me = await applicantSession();
  if (!me) {
    location.replace('/track'); // без сессии заявителя смотреть обращение нельзя
    return;
  }
  await loadApplicantView();
  viewPoller.start();
}
