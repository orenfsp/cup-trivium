import { $, esc, fmtTime, toast, showErr, hideErr } from '../core/dom.js';
import { api } from '../core/api.js';
import { registerActions } from '../core/actions.js';
import { t } from './tone.js';

let lastTrack = '';

export const rememberTrack = (t) => { lastTrack = t; };

export const lastTrackNumber = () => lastTrack;

// Прямая ссылка на обращение — отдельный эндпоинт /appeal с трек-номером в параметре.
const trackURL = (t) => location.origin + '/appeal?track=' + encodeURIComponent(t);

export function saveTrack(track) {
  const saved = JSON.parse(localStorage.getItem('otklik_tracks') || '[]');
  if (saved.some((s) => s.track === track)) return;
  saved.unshift({ track, created: new Date().toISOString() });
  localStorage.setItem('otklik_tracks', JSON.stringify(saved.slice(0, 10)));
  renderSaved();
}

function copyTrack() {
  const t = lastTrack || $('apTrack').textContent.trim();
  if (!t) return;
  const done = () => toast('Трек-номер скопирован');
  if (navigator.clipboard && navigator.clipboard.writeText) {
    navigator.clipboard.writeText(t).then(done).catch(() => fallbackCopy(t, done));
  } else fallbackCopy(t, done);
}

function fallbackCopy(t, done) {
  const ta = document.createElement('textarea');
  ta.value = t;
  ta.style.position = 'fixed';
  ta.style.opacity = '0';
  document.body.appendChild(ta);
  ta.select();
  try { document.execCommand('copy'); done(); }
  catch (e) { toast('Скопируйте номер вручную: ' + t); }
  ta.remove();
}

function downloadTrack() {
  const t = lastTrack || $('apTrack').textContent.trim();
  if (!t) return;
  const txt =
    'Сервис «Отклик» — трек-номер обращения\n\n' +
    t + '\n\n' +
    'Сохрани этот файл или распечатай его.\n' +
    'Открыть своё обращение можно по ссылке:\n' + trackURL(t) + '\n' +
    'или на сайте сервиса: «Вход по трек-номеру».\n' +
    'Сервис анонимный: трек-номер — единственный способ вернуться к обращению.\n' +
    'ВНИМАНИЕ: если потеряешь номер — восстановить его будет невозможно!\n' +
    'Детский телефон доверия: 8-800-2000-122 (бесплатно, круглосуточно).';
  const a = document.createElement('a');
  a.href = URL.createObjectURL(new Blob(['\ufeff' + txt], { type: 'text/plain;charset=utf-8' }));
  a.download = 'Отклик ' + t + '.txt';
  document.body.appendChild(a);
  a.click();
  a.remove();
  setTimeout(() => URL.revokeObjectURL(a.href), 5000);
}

export function renderQR(track) {
  const box = $('apQR');
  box.innerHTML = '';
  if (typeof qrcode === 'undefined') return; // библиотека QR (локальная, /assets/vendor) не загрузилась — просто без кода
  try {
    const qr = qrcode(0, 'M');
    qr.addData(trackURL(track));
    qr.make();
    box.innerHTML = `<img src="${qr.createDataURL(5, 8)}" width="200" height="200" alt="QR-код для входа по трек-номеру">` +
      '<div class="note">Наведите камеру телефона — откроется ваше обращение</div>';
  } catch (e) {
    box.innerHTML = '';
  }
}

export function renderSaved() {
  const box = $('savedTracks'); // список есть только на странице /track
  if (!box) return;
  const saved = JSON.parse(localStorage.getItem('otklik_tracks') || '[]');
  box.innerHTML = saved.length
    ? saved.map((s) => `<div style="margin:6px 0"><a href="#" data-action="use-track" data-arg="${esc(s.track)}">${esc(s.track)}</a> <span class="note">${fmtTime(s.created)}</span></div>`).join('')
    : 'пусто';
}

function useTrack(t) {
  $('trkInput').value = t;
  verifyTrack();
}

export async function verifyTrack() {
  hideErr('trkErr');
  const tn = $('trkInput').value.trim();
  if (!tn) { showErr('trkErr', new Error(t('Вставь трек-номер обращения — он в файле-памятке или в QR-коде', 'Вставьте трек-номер обращения — он в файле-памятке или в QR-коде'))); return; }
  try {
    await api('POST', '/api/appeals/track', { track_number: tn });
    saveTrack(tn);
    location.href = '/appeal'; // обращение живёт на отдельном эндпоинте
  } catch (e) {
    showErr('trkErr', e);
  }
}

registerActions({
  'copy-track': copyTrack,
  'download-track': downloadTrack,
  'verify-track': verifyTrack,
  'use-track': useTrack,
});
