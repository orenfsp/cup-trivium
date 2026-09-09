export const staffState = { me: null };

// Домашняя страница сотрудника по роли — на неё редиректим после входа и в гардах страниц.
export const roleHome = (role) =>
  role === 'admin' ? '/admin'
    : role === 'expert' ? '/expert'
      : role === 'operator' ? '/operator'
        : '/login';
