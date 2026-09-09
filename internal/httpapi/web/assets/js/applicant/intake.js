import { $, esc, showErr, hideErr } from '../core/dom.js';
import { api } from '../core/api.js';
import { registerActions } from '../core/actions.js';
import { loadCategories } from '../categories.js';
import { rememberTrack, saveTrack, renderQR } from './track.js';
import { applyTone, t } from './tone.js';

function syncFree() {
  $('apCat').disabled = $('apFree').checked;
  $('apCat').parentElement.style.opacity = $('apFree').checked ? 0.5 : 1;
}

async function renderQuestions() {
  try {
    const qs = (await api('GET', '/api/intake-questions')).questions || [];
    $('apQuestions').innerHTML = qs.map((q) =>
      `<div><label>${esc(q.text)}</label>
       <select data-q="${esc(q.text)}">
         <option value="">— пропустить</option>
         ${(q.options || []).map((o) => `<option value="${esc(o)}">${esc(o)}</option>`).join('')}
       </select></div>`).join('');
  } catch (e) {
    console.error(e);
  }
}

function renderCrisisHelp(help) {
  const box = $('apCrisisHelp');
  if (!help || !help.length) { box.classList.add('hidden'); box.innerHTML = ''; return; }
  box.innerHTML = '<b>⚠ Кризисная помощь — позвоните прямо сейчас:</b>' +
    help.map((h) => {
      const digits = String(h.phone).replace(/\D/g, '');
      const tel = (digits.length === 11 && digits[0] === '8') ? '+7' + digits.slice(1) : digits;
      return `<div class="crisis-call"><div class="crisis-txt"><div class="crisis-t">${esc(h.title)}</div><div class="crisis-d">${esc(h.description || '')}</div></div>` +
        `<a class="crisis-tel" href="tel:${tel}" aria-label="Позвонить: ${esc(h.title)}">📞 ${esc(h.phone)}</a></div>`;
    }).join('');
  box.classList.remove('hidden');
}

async function createAppeal() {
  hideErr('apErr');
  const desc = $('apDesc').value.trim();
  if (desc.length < 10) {
    showErr('apErr', new Error(t('Можешь добавить пару деталей? Так будет проще помочь', 'Можете добавить пару деталей? Так будет проще помочь')));
    return;
  }
  const body = {
    applicant_type: $('apType').value,
    free_text: $('apFree').checked,
    description: desc,
    crisis_contact: $('apCrisis').value.trim(),
    answers: [...document.querySelectorAll('#apQuestions select')]
      .map((s) => ({ question: s.dataset.q, answer: s.value }))
      .filter((a) => a.answer)
  };
  if (!body.free_text) {
    if (!$('apCat').value) {
      showErr('apErr', new Error(t('Выбери категорию — так мы скорее найдём подходящего специалиста', 'Выберите категорию — так мы скорее найдём подходящего специалиста')));
      return;
    }
    body.category_id = $('apCat').value;
  }
  try {
    const r = await api('POST', '/api/appeals', body);
    rememberTrack(r.track_number);
    $('apTrack').textContent = r.track_number;
    $('apResult').classList.remove('hidden');
    $('apResult').scrollIntoView({ behavior: 'smooth', block: 'center' });
    $('apCrisisNote').textContent = r.crisis_detected
      ? t('Похоже, сейчас тебе непросто. Если поддержка нужна срочно — обратись в службу из списка ниже: они работают прямо сейчас.',
          'Похоже, ситуация серьёзная. Если поддержка нужна срочно — обратитесь в службу из списка ниже: они работают прямо сейчас.')
      : '';
    // ТЗ «Кризисные обращения», п.2: полная кризисная помощь сразу, не дожидаясь оператора.
    renderCrisisHelp(r.crisis_help);
    saveTrack(r.track_number);
    renderQR(r.track_number);
    $('trkInput').value = r.track_number;
  } catch (e) {
    showErr('apErr', e);
  }
}

export async function intakeInit() {
  $('apFree').addEventListener('change', syncFree);
  $('apType').addEventListener('change', () => applyTone($('apType').value));
  applyTone($('apType').value);
  await loadCategories();
  await renderQuestions();
  syncFree();
}

registerActions({
  'create-appeal': createAppeal,
  'goto-new': () => $('newAppealCard').scrollIntoView({ behavior: 'smooth', block: 'start' }),
  'goto-track': () => $('trkInput').closest('.card').scrollIntoView({ behavior: 'smooth', block: 'start' }),
});
