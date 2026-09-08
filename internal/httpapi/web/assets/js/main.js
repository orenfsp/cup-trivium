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
