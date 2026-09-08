// Точка входа UI. Режим (applicant/staff) подставляется сервером при отдаче
// страницы — это два разных порта с разными наборами API-маршрутов, поэтому
// и код профиля загружается динамическим импортом: каждый порт получает
// только свои модули.
import { bindActionDelegation } from './core/actions.js';

bindActionDelegation();

(async () => {
  if (window.UI_MODE === 'staff') {
    const { staffInit } = await import('./staff/index.js');
    await staffInit();
  } else {
    const { applicantInit } = await import('./applicant/index.js');
    await applicantInit();
  }
})();
