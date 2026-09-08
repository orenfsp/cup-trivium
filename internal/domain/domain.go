// Package domain содержит общие доменные типы: роли, статусы,
package domain

import (
	"errors"

	"github.com/google/uuid"
)

// Роли субъектов системы.
type Role string

const (
	RoleApplicant Role = "applicant" // анонимный заявитель (сессия по трек-номеру)
	RoleOperator  Role = "operator"  // оператор
	RoleExpert    Role = "expert"    // эксперт
	RoleAdmin     Role = "admin"     // администратор
)

// Статусы обращения.
type Status string

const (
	StatusNew                Status = "new"
	StatusAssigned           Status = "assigned"
	StatusInProgress         Status = "in_progress"
	StatusNeedsClarification Status = "needs_clarification"
	StatusAnswerReady        Status = "answer_ready"
	StatusCompleted          Status = "completed"
	StatusReturned           Status = "returned"
	StatusRejected           Status = "rejected"
	StatusClosedNoResponse   Status = "closed_no_response"
)

var allStatuses = map[Status]bool{
	StatusNew: true, StatusAssigned: true, StatusInProgress: true,
	StatusNeedsClarification: true, StatusAnswerReady: true, StatusCompleted: true,
	StatusReturned: true, StatusRejected: true, StatusClosedNoResponse: true,
}

func (s Status) Valid() bool { return allStatuses[s] }

func (s Status) Terminal() bool {
	return s == StatusCompleted || s == StatusRejected || s == StatusClosedNoResponse
}

var Transitions = map[Status]map[Status]bool{
	StatusNew: {
		StatusAssigned:  true,
		StatusRejected:  true,
		StatusCompleted: true, // оператор помог самостоятельно
	},
	StatusAssigned: {
		StatusInProgress: true,
	},
	StatusInProgress: {
		StatusNeedsClarification: true,
		StatusAnswerReady:        true,
	},
	StatusNeedsClarification: {
		StatusInProgress:    true,
		StatusClosedNoResponse: true, // заявитель не ответил
	},
	StatusAnswerReady: {
		StatusCompleted:       true,
		StatusReturned:        true,
		StatusClosedNoResponse: true, // заявитель не подтвердил результат
	},
	StatusReturned: {
		StatusAssigned: true,
		StatusRejected: true,
	},
}

func CanTransition(from, to Status) bool {
	if from == to || !to.Valid() {
		return false
	}
	allowed, ok := Transitions[from]
	return ok && allowed[to]
}

// Приоритет обращения.
type Priority string

const (
	PriorityLow    Priority = "low"
	PriorityNormal Priority = "normal"
	PriorityUrgent Priority = "urgent"
)

func (p Priority) Valid() bool {
	return p == PriorityLow || p == PriorityNormal || p == PriorityUrgent
}

// Тип заявителя.
type ApplicantType string

const (
	ApplicantSchoolchild ApplicantType = "schoolchild"
	ApplicantParent      ApplicantType = "parent"
	ApplicantTeacher     ApplicantType = "teacher"
)

func (t ApplicantType) Valid() bool {
	return t == ApplicantSchoolchild || t == ApplicantParent || t == ApplicantTeacher
}


type Principal struct {
	Role     Role
	UserID   uuid.UUID // для сотрудников
	Login    string    // для сотрудников
	AppealID uuid.UUID // только для заявителя
}

func (p Principal) IsStaff() bool {
	return p.Role == RoleOperator || p.Role == RoleExpert || p.Role == RoleAdmin
}

// Ошибки домена.
var (
	ErrUnauthorized = errors.New("unauthorized")
	ErrForbidden    = errors.New("forbidden")
	ErrNotFound     = errors.New("not found")
	ErrConflict     = errors.New("conflict")
	ErrValidation   = errors.New("validation error")
	ErrRateLimited  = errors.New("rate limited")
)


type IntakeQuestion struct {
	Text    string   `json:"text"`
	Options []string `json:"options"`
}

var IntakeQuestions = []IntakeQuestion{
	{Text: "Где это происходит?", Options: []string{
		"В школе", "В интернете или соцсетях", "И там, и там", "Другое место"}},
	{Text: "Как давно это происходит?", Options: []string{
		"Только что", "Несколько дней", "Несколько недель", "Больше месяца"}},
	{Text: "Кто участвует?", Options: []string{
		"Одноклассники", "Учитель", "Родители или семья", "Незнакомые люди", "Несколько людей"}},
	{Text: "Обращался ли кто-то уже за помощью?", Options: []string{
		"Нет, это первое обращение", "Да, говорили взрослым", "Да, уже обращались сюда"}},
}

// MaxAttachmentsPerAppeal — максимум вложений на обращение.
const MaxAttachmentsPerAppeal = 5

// MaxAttachmentSizeBytes — лимит одного файла.
const MaxAttachmentSizeBytes = 10 << 20 // 10 МБ на файл

// Пороги контроля SLA, в часах: обращение в очереди новых,
// ждущее обработки дольше QueueOverdueHours, — просрочено; распределённое
// обращение, где заявитель не получил ответа дольше ResponseOverdueHours,
// считается зависшим.
const (
	QueueOverdueHours    = 24
	ResponseOverdueHours = 24
)
