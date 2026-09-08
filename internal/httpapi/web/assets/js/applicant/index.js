import { $ } from '../core/dom.js';
import { api } from '../core/api.js';
import { intakeInit } from './intake.js';
import { saveTrack, renderSaved, verifyTrack } from './track.js';
import { loadApplicantView, viewPoller } from './view.js';

export async function applicantInit() {
  await intakeInit();
  renderSaved();
  const urlTrack = new URLSearchParams(location.search).get('track');
  if (urlTrack) {
    saveTrack(urlTrack);
    $('trkInput').value = urlTrack;
    history.replaceState(null, '', location.pathname); // убираем номер из адресной строки
    verifyTrack();
  }
  try {
    const r = await api('GET', '/api/me');
    if (r.role === 'applicant') {
      $('whoami').textContent = 'заявитель';
      await loadApplicantView();
      viewPoller.start();
    }
  } catch (e) { /* нет активной сессии заявителя — нормальная ситуация */ }
}
