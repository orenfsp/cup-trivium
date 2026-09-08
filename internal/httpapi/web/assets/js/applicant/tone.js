let currentTone = 'formal';

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

export const t = (informal, formal) => (currentTone === 'informal' ? informal : formal);
