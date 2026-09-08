// Списки обращений сотрудника: очередь оператора (требуют действий),
// общий список оператора с фильтром по статусу и список назначенного эксперту.
import { $, esc, fmtTime, badge, toast } from '../core/dom.js';
import { api } from '../core/api.js';
import { ruStatus, ruPrio } from '../core/i18n.js';
import { registerActions } from '../core/actions.js';
import { staffState } from './state.js';

// Карточка-элемент списка обращений; у операторских списков есть флаг
// transfer_requested — на карточке сразу видно, что специалист просит замену.
export function appealItem(a, extra) {
  // Срочные всегда сверху (сортирует сервер) и визуально
  // выделены рамкой/фоном карточки.
  return `<div class="item${a.priority === 'urgent' ? ' urgent' : ''}" data-action="open-detail" data-arg="${esc(a.id)}">
    <div class="l1"><span class="cat">${esc(a.category_name || 'Свободный текст')}</span>${badge(a.status, ruStatus(a.status))}${a.transfer_requested ? badge('transfer', '↔ требуется другой специалист') : ''}</div>
    <div class="l2">${badge(a.priority, ruPrio(a.priority))}${a.crisis_detected ? badge('crisis', '⚠ кризис') : ''}
      <span>#${esc(String(a.id).slice(0, 8))}</span>${extra || ''}</div>
  </div>`;
}

/** Длительность в человекочитаемом виде: «2 ч 15 мин», «40 мин». */
const fmtDur = (sec) => {
  if (sec == null) return '—';
  const h = Math.floor(sec / 3600), m = Math.round((sec % 3600) / 60);
  return h > 0 ? (m ? `${h} ч ${m} мин` : `${h} ч`) : `${Math.max(1, m)} мин`;
};

export async function loadQueue() {
  try {
    const q = (await api('GET', '/api/operator/queue')).appeals || [];
    // Счётчик просроченных — тех, что ждут обработки дольше SLA.
    const overdue = q.filter((a) => a.overdue).length;
    const cnt = $('opQueueOverdue');
    cnt.textContent = `просроченных: ${overdue}`;
    cnt.classList.toggle('hidden', overdue === 0);
    $('opQueue').innerHTML = q.map((a) =>
      appealItem(a, `<span>ожидание ${fmtDur(a.waiting_sec)}</span>${a.overdue ? badge('overdue', '⏱ просрочено') : ''}` +
        // Особые случаи маршрутизации видны оператору прямо в очереди.
        (a.no_expert_in_group ? badge('crisis', 'нет специалистов группы') : '') +
        (a.group_overloaded ? badge('transfer', 'группа перегружена') : ''))
    ).join('') || '<div class="note" style="margin-top:8px">очередь пуста</div>';
  } catch (e) { toast(e.message); }
}

// Общий список обращений оператора: видно статус и ответственного.
// В фильтре «распределённые» зависшие без ответа дольше SLA отмечаются
// бейджем; текст переписки в списке не показывается.
export async function loadOpAll() {
  try {
    const st = $('opFilterStatus').value;
    const list = (await api('GET', '/api/operator/appeals' + (st ? '?status=' + st : ''))).appeals || [];
    $('opAll').innerHTML = list.map((a) =>
      appealItem(a, `<span>${esc(a.assigned_expert || 'не назначен')}</span><span>обновлено ${fmtTime(a.updated_at)}</span>` +
        (a.no_reply_sec != null && a.no_reply_sec > 24 * 3600 ? badge('overdue', `⏱ без ответа ${fmtDur(a.no_reply_sec)}`) : ''))
    ).join('') || '<div class="note" style="margin-top:8px">нет обращений</div>';
  } catch (e) { toast(e.message); }
}

// «Мои обращения» — только назначенные эксперту; фильтры по статусу,
// приоритету и категории, срочные — наверху.
export async function loadExpert() {
  try {
    const qs = new URLSearchParams();
    if ($('exFilterStatus').value) qs.set('status', $('exFilterStatus').value);
    if ($('exFilterPriority').value) qs.set('priority', $('exFilterPriority').value);
    if ($('exFilterCategory').value) qs.set('category', $('exFilterCategory').value);
    const q = qs.toString();
    const list = (await api('GET', '/api/expert/appeals' + (q ? '?' + q : ''))).appeals || [];
    $('exList').innerHTML = list.map((a) => appealItem(a)).join('') || '<div class="note" style="margin-top:8px">нет обращений</div>';
  } catch (e) { toast(e.message); }
}

/** Единожды подключить слушатели фильтров-селектов (событие change). */
export function bindListFilters() {
  $('opFilterStatus').addEventListener('change', loadOpAll);
  $('exFilterStatus').addEventListener('change', loadExpert);
  $('exFilterPriority').addEventListener('change', loadExpert);
  $('exFilterCategory').addEventListener('change', loadExpert);
  // Список активных категорий для фильтра эксперта — общий эндпоинт.
  api('GET', '/api/categories').then((r) => {
    $('exFilterCategory').innerHTML = '<option value="">любая</option>' +
      (r.categories || []).map((c) => `<option value="${esc(c.name)}">${esc(c.name)}</option>`).join('');
  }).catch(() => {});
}

// Аналитика «по себе»: оператор видит результат своих назначений,
// специалист — нагрузку по своим обращениям.
export async function loadMyStats() {
  const me = staffState.me;
  if (!me || (me.role !== 'operator' && me.role !== 'expert')) return;
  const el = me.role === 'expert' ? $('exMyStats') : $('opMyStats');
  try {
    const s = await api('GET', '/api/mystats');
    const row = (l, v) => `<div class="kv"><b>${l}</b><span>${v}</span></div>`;
    const avg = s.avg_resolution_hours != null ? (Math.round(s.avg_resolution_hours * 10) / 10) + ' ч' : '—';
    el.innerHTML = me.role === 'operator'
      ? row('Назначено обращений', s.assigned) +
        row('В работе из назначенных', s.active) +
        row('Завершено из назначенных', s.resolved) +
        row('Отклонено мной', s.rejected)
      : row('Активных обращений', s.active) +
        row('Завершено', s.resolved) +
        row('Опубликовано рекомендаций', s.recommendations) +
        row('Среднее время решения', avg);
  } catch (e) { el.textContent = e.message; }
}

// Выгрузка CSV: оператор/админ — все обращения, специалист — только свои.
const exportCsv = () => { window.open('/api/export/appeals', '_blank'); };

registerActions({
  'reload-queue': loadQueue,
  'reload-op-all': loadOpAll,
  'reload-expert': loadExpert,
  'reload-my-stats': loadMyStats,
  'export-csv': exportCsv,
});
