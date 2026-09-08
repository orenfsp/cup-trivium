// Тон общения (ТЗ п.3.1): со школьником — на «ты», с родителем и педагогом —
// на «вы». Тексты переключаются вместе с выбором типа заявителя: элементы
// несут пары data-informal / data-formal (и ...-ph для placeholder'ов),
// а код выбирает нужную форму через t(...).

let currentTone = 'formal';

/** Применить тон ко всем разметанным в HTML текстам. */
export function applyTone(applicantType) {
  currentTone = applicantType === 'schoolchild' ? 'informal' : 'formal';
  document.querySelectorAll('[data-informal]').forEach((el) => {
    el.textContent = el.dataset[currentTone] || el.textContent;
  });
  document.querySelectorAll('[data-informal-ph]').forEach((el) => {
    const v = el.dataset[currentTone + 'Ph'];
    if (v != null) el.setAttribute('placeholder', v);
  });
}

/** Выбрать форму фразы по текущему тону. */
export const t = (informal, formal) => (currentTone === 'informal' ? informal : formal);
