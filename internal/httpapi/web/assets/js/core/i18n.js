const RU_STATUS = { new: 'новое', assigned: 'назначено', in_progress: 'в работе', needs_clarification: 'нужны уточнения', answer_ready: 'готов ответ', completed: 'завершено', returned: 'возвращено', rejected: 'отклонено', closed_no_response: 'закрыто без ответа' };
const RU_PRIO = { low: 'низкий', normal: 'обычный', urgent: 'срочный' };
const RU_ROLE = { applicant: 'заявитель', operator: 'оператор', expert: 'специалист', admin: 'администратор' };
const RU_GROUP = { psychologists: 'психолог', conflictologists: 'конфликтолог', lawyers: 'юрист', social_pedagogues: 'социальный педагог', mediators: 'конфликтолог' };
const RU_APPTYPE = { schoolchild: 'школьник', parent: 'родитель', teacher: 'педагог' };
const RU_AUTHOR = { applicant: 'заявитель', operator: 'оператор', expert: 'специалист', system: 'система' };
const RU_EVENT = { status: 'статус', priority: 'приоритет', category: 'категория', assign: 'назначение эксперта', transfer_requested: 'запрос передачи', contributor_added: 'добавлен соисполнитель', append: 'дополнение от заявителя', crisis: '⚠ кризис-маркеры' };

const localize = (dict) => (s) => dict[s] || s;

export const ruStatus = localize(RU_STATUS);
export const ruPrio = localize(RU_PRIO);
export const ruRole = localize(RU_ROLE);
export const ruGroup = localize(RU_GROUP);
export const ruAppType = localize(RU_APPTYPE);
export const ruAuthor = localize(RU_AUTHOR);
export const ruEvent = localize(RU_EVENT);
