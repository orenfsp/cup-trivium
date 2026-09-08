import { $, esc, fmtTime, badge, kv, toast } from '../core/dom.js';
import { api } from '../core/api.js';
import { ruStatus, ruAuthor } from '../core/i18n.js';
import { createPoller } from '../core/poller.js';
import { registerActions } from '../core/actions.js';
import { applyTone, t } from './tone.js';

export const viewPoller = createPoller(loadApplicantView, 5000);

const TERMINAL = ['completed', 'rejected', 'closed_no_response'];

const fmtSize = (n) => n >= 1048576 ? (n / 1048576).toFixed(1) + ' МБ' : Math.max(1, Math.round(n / 1024)) + ' КБ';

export async function loadApplicantView() {
  let v;
  try { v = await api('GET', '/api/appeals/me/'); } catch (e) { return; }
  applyTone(v.applicant_type);
  $('apViewCard').classList.remove('hidden');
  $('avStatus').innerHTML = badge(v.status, ruStatus(v.status));
  $('avExpl').textContent = v.status_explanation || '';
  $('avExpl').classList.toggle('hidden', !v.status_explanation);
  $('avCrisis').classList.toggle('hidden', !v.crisis_help);
  if (v.crisis_help) {
    $('avCrisis').innerHTML = '<b>⚠ Кризисная помощь — позвоните прямо сейчас:</b>' +
      v.crisis_help.map((h) => {
        const digits = String(h.phone).replace(/\D/g, '');
        const tel = (digits.length === 11 && digits[0] === '8') ? '+7' + digits.slice(1) : digits;
        return `<div class="crisis-call"><div class="crisis-txt"><div class="crisis-t">${esc(h.title)}</div><div class="crisis-d">${esc(h.description || '')}</div></div>` +
          `<a class="crisis-tel" href="tel:${tel}" aria-label="Позвонить: ${esc(h.title)}">📞 ${esc(h.phone)}</a></div>`;
      }).join('');
  }
  $('avMeta').innerHTML =
    kv('Категория', esc(v.category_name || 'свободный текст')) +
    kv('Создано', fmtTime(v.created_at)) +
    kv('Возвратов на доработку', v.return_count) +
    (v.recommendation ? kv('Рекомендация', esc(v.recommendation)) : '');
  $('avDesc').textContent = v.description || '';
  const closed = TERMINAL.includes(v.status);
  $('avAppendBlock').classList.toggle('hidden', closed);
  $('avUploadBlock').classList.toggle('hidden', closed);
  $('avAttach').innerHTML = (v.attachments || []).map((a) =>
    `<div class="kv"><b>${esc(a.content_type)}</b><span><a href="/api/appeals/me/attachments/${a.id}" target="_blank" rel="noopener">открыть файл</a> · ${fmtSize(a.size)} · ${fmtTime(a.created_at)}</span></div>`
  ).join('') || '<div class="note">пока нет вложений</div>';
  $('avAnswers').innerHTML = (v.intake_answers || []).map((a) =>
    `<div class="kv"><b>${esc(a.question)}</b><span>${esc(a.answer)}</span></div>`).join('') || '<div class="note">—</div>';
  $('avHistory').innerHTML = (v.status_history || []).map((h) => {
    const st = h.status || h.to_status;
    return `<li>${badge(st, ruStatus(st))} — ${fmtTime(h.created_at || h.at)}</li>`;
  }).join('');
  $('avMsgs').innerHTML = (v.messages || []).map((m) =>
    `<div class="msg ${m.author_type === 'applicant' ? 'mine' : ''}"><div class="a">${esc(ruAuthor(m.author_type))} · ${fmtTime(m.created_at)}</div>${esc(m.text)}</div>`).join('')
    || '<div class="note">пока нет сообщений</div>';
  $('avMsgs').scrollTop = 1e9;
  $('avResultBlock').classList.toggle('hidden', v.status !== 'answer_ready');
  $('avFeedbackBlock').classList.toggle('hidden', !(v.status === 'answer_ready' || v.status === 'completed'));
  $('avAgainBlock').classList.toggle('hidden', !closed);
}

async function apSendMessage() {
  const t = $('avMsgText').value.trim();
  if (!t) return;
  try {
    await api('POST', '/api/appeals/me/messages', { text: t });
    $('avMsgText').value = '';
    loadApplicantView();
  } catch (e) { toast(e.message); }
}

async function apAppend() {
  const text = $('avAppendText').value.trim();
  if (text.length < 10) {
    toast(t('Добавь хотя бы пару слов — так специалисту будет понятнее', 'Добавьте хотя бы пару слов — так специалисту будет понятнее'));
    return;
  }
  try {
    await api('POST', '/api/appeals/me/append', { text });
    $('avAppendText').value = '';
    toast(t('Дополнение добавлено — специалист его увидит', 'Дополнение добавлено — специалист его увидит'));
    loadApplicantView();
  } catch (e) { toast(e.message); }
}

async function apUpload() {
  const files = [...$('avFile').files];
  if (!files.length) { toast(t('Сначала выбери файл(ы)', 'Сначала выберите файл(ы)')); return; }
  let uploaded = 0;
  for (const f of files) {
    if (f.size > 10 * 1024 * 1024) {
      toast(t(`«${f.name}» больше 10 МБ — приложить не получится`, `Файл «${f.name}» больше 10 МБ — приложить не получится`));
      continue;
    }
    const fd = new FormData();
    fd.append('file', f);
    const res = await fetch('/api/appeals/me/attachments', { method: 'POST', body: fd });
    if (!res.ok) {
      const d = await res.json().catch(() => null);
      toast((d && d.error) || ('HTTP ' + res.status));
      continue;
    }
    uploaded++;
  }
  $('avFile').value = '';
  if (uploaded) toast(t('Файлы прикреплены', 'Файлы прикреплены'));
  loadApplicantView();
}

async function apResult(helped) {
  const reason = helped ? '' : $('avReturnReason').value.trim();
  try {
    await api('POST', '/api/appeals/me/result', { helped, reason });
    loadApplicantView();
    toast(helped
      ? t('Спасибо! Рады, что помогло', 'Спасибо! Отмечено как полезное')
      : t('Спасибо за честный ответ — подберём другой способ помочь', 'Спасибо за отзыв — подберём другой способ помощи'));
  } catch (e) { toast(e.message); }
}

async function apFeedback() {
  try {
    await api('POST', '/api/appeals/me/feedback', { rating: +$('avRating').value, comment: $('avComment').value.trim() || null });
    toast('Оценка сохранена');
  } catch (e) { toast(e.message); }
}

registerActions({
  'applicant-send-message': apSendMessage,
  'applicant-append': apAppend,
  'applicant-upload': apUpload,
  'applicant-result': (arg) => apResult(arg === '1'),
  'applicant-feedback': apFeedback,
  'write-again': () => {
    $('newAppealCard').scrollIntoView({ behavior: 'smooth', block: 'start' });
    $('apDesc').focus();
  },
});
