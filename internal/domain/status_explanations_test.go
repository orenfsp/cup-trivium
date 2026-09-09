package domain

import (
	"strings"
	"testing"

	"github.com/google/uuid"
)

func everyStatus() []Status {
	return []Status{
		StatusNew, StatusAssigned, StatusInProgress, StatusNeedsClarification,
		StatusAnswerReady, StatusCompleted, StatusReturned, StatusRejected, StatusClosedNoResponse,
	}
}

// ТЗ 4.4: каждый статус объясняется заявителю просто, в двух тональностях.
func TestExplanationForEveryStatus(t *testing.T) {
	for _, s := range everyStatus() {
		informal := s.ExplanationFor(ApplicantSchoolchild)
		formal := s.ExplanationFor(ApplicantParent)
		if strings.TrimSpace(informal) == "" {
			t.Errorf("%s: пустое объяснение на «ты»", s)
		}
		if strings.TrimSpace(formal) == "" {
			t.Errorf("%s: пустое объяснение на «вы»", s)
		}
		if informal == formal {
			t.Errorf("%s: тональности не должны совпадать", s)
		}
	}
}

// ТЗ 5.1: при отказе заявитель получает альтернативу — телефон доверия.
func TestExplanationRejectedHasHelpline(t *testing.T) {
	for _, at := range []ApplicantType{ApplicantSchoolchild, ApplicantParent, ApplicantTeacher} {
		if !strings.Contains(StatusRejected.ExplanationFor(at), "8-800-2000-122") {
			t.Errorf("отказ для %s должен содержать телефон доверия", at)
		}
	}
}

func TestExplanationUnknownStatusEmpty(t *testing.T) {
	if got := Status("hacked").ExplanationFor(ApplicantSchoolchild); got != "" {
		t.Errorf("неизвестный статус должен давать пустое объяснение, got %q", got)
	}
}

func TestStatusValid(t *testing.T) {
	for _, s := range everyStatus() {
		if !s.Valid() {
			t.Errorf("Status %s should be valid", s)
		}
	}
	for _, s := range []Status{"", "unknown", "NEW"} {
		if s.Valid() {
			t.Errorf("Status %q should be invalid", s)
		}
	}
}

func TestPriorityValid(t *testing.T) {
	for _, p := range []Priority{PriorityLow, PriorityNormal, PriorityUrgent} {
		if !p.Valid() {
			t.Errorf("Priority %s should be valid", p)
		}
	}
	for _, p := range []Priority{"", "high", "URGENT"} {
		if p.Valid() {
			t.Errorf("Priority %q should be invalid", p)
		}
	}
}

func TestApplicantTypeValid(t *testing.T) {
	for _, at := range []ApplicantType{ApplicantSchoolchild, ApplicantParent, ApplicantTeacher} {
		if !at.Valid() {
			t.Errorf("ApplicantType %s should be valid", at)
		}
	}
	for _, at := range []ApplicantType{"", "alien", "PARENT"} {
		if at.Valid() {
			t.Errorf("ApplicantType %q should be invalid", at)
		}
	}
}

func TestPrincipalIsStaff(t *testing.T) {
	staff := []Role{RoleOperator, RoleExpert, RoleAdmin}
	for _, r := range staff {
		if !(Principal{Role: r}).IsStaff() {
			t.Errorf("Role %s should be staff", r)
		}
	}
	if (Principal{Role: RoleApplicant, AppealID: uuid.New()}).IsStaff() {
		t.Error("applicant should not be staff")
	}
}

// ТЗ: 4 необязательных уточняющих вопроса с вариантами ответа.
func TestIntakeQuestionsPerSpec(t *testing.T) {
	if len(IntakeQuestions) != 4 {
		t.Fatalf("expected 4 intake questions, got %d", len(IntakeQuestions))
	}
	for _, q := range IntakeQuestions {
		if strings.TrimSpace(q.Text) == "" {
			t.Errorf("пустой текст вопроса: %+v", q)
		}
		if len(q.Options) < 3 {
			t.Errorf("вопрос %q должен иметь минимум 3 варианта, got %d", q.Text, len(q.Options))
		}
		for _, o := range q.Options {
			if strings.TrimSpace(o) == "" {
				t.Errorf("пустой вариант в вопросе %q", q.Text)
			}
		}
	}
}

// ТЗ «Кризисные обращения»: справочник контактов должен быть заполнен
// и содержать детский телефон доверия.
func TestCrisisHelpContacts(t *testing.T) {
	if len(CrisisHelpContacts) < 3 {
		t.Fatalf("expected at least 3 crisis contacts, got %d", len(CrisisHelpContacts))
	}
	hasHelpline := false
	for _, c := range CrisisHelpContacts {
		if c.Title == "" || c.Phone == "" || c.Description == "" {
			t.Errorf("неполный контакт: %+v", c)
		}
		if c.Phone == "8-800-2000-122" {
			hasHelpline = true
		}
	}
	if !hasHelpline {
		t.Error("справочник должен содержать детский телефон доверия 8-800-2000-122")
	}
}

// Нормализация кризисного текста: ё, регистр и переводы строк не мешают детекту.
func TestDetectCrisisNormalization(t *testing.T) {
	if !DetectCrisis("  ВСЁ   НАДОЕЛО\nпо-настоящему ") {
		t.Error("DetectCrisis должен находить маркер при лишних пробелах и регистре")
	}
}
