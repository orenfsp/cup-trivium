package domain

import (
	"regexp"
	"testing"
)

func TestCanTransition(t *testing.T) {
	allowed := []struct{ from, to Status }{
		{StatusNew, StatusAssigned},
		{StatusNew, StatusRejected},
		{StatusNew, StatusCompleted},
		{StatusAssigned, StatusInProgress},
		{StatusInProgress, StatusNeedsClarification},
		{StatusInProgress, StatusAnswerReady},
		{StatusNeedsClarification, StatusInProgress},
		{StatusNeedsClarification, StatusClosedNoResponse},
		{StatusAnswerReady, StatusCompleted},
		{StatusAnswerReady, StatusReturned},
		{StatusAnswerReady, StatusClosedNoResponse},
		{StatusReturned, StatusAssigned},
		{StatusReturned, StatusRejected},
	}
	for _, tc := range allowed {
		if !CanTransition(tc.from, tc.to) {
			t.Errorf("CanTransition(%s, %s) = false, want true", tc.from, tc.to)
		}
	}

	forbidden := []struct{ from, to Status }{
		{StatusNew, StatusInProgress},       // минуя назначение
		{StatusAssigned, StatusAnswerReady}, // минуя работу
		{StatusCompleted, StatusNew},        // терминальный статус
		{StatusRejected, StatusAssigned},    // терминальный статус
		{StatusClosedNoResponse, StatusInProgress},
		{StatusInProgress, StatusInProgress}, // самопереход
		{StatusNew, Status("hacked")},        // неизвестный статус
	}
	for _, tc := range forbidden {
		if CanTransition(tc.from, tc.to) {
			t.Errorf("CanTransition(%s, %s) = true, want false", tc.from, tc.to)
		}
	}
}

func TestTerminal(t *testing.T) {
	terminal := []Status{StatusCompleted, StatusRejected, StatusClosedNoResponse}
	for _, s := range terminal {
		if !s.Terminal() {
			t.Errorf("Status %s should be terminal", s)
		}
	}
	for _, s := range []Status{StatusNew, StatusAssigned, StatusInProgress, StatusNeedsClarification, StatusAnswerReady, StatusReturned} {
		if s.Terminal() {
			t.Errorf("Status %s should not be terminal", s)
		}
	}
}

func TestDetectCrisis(t *testing.T) {
	positive := []string{
		"Мне не хочу жить",
		"ХОЧУ УМЕРЕТЬ", // регистр не важен
		"Он сказал, что убьёт меня",
		"Родители избивают",
		"всё надоело",
	}
	for _, s := range positive {
		if !DetectCrisis(s) {
			t.Errorf("DetectCrisis(%q) = false, want true", s)
		}
	}
	if DetectCrisis("Всё хорошо, просто спросить про расписание") {
		t.Error("DetectCrisis should be false for neutral text")
	}
	// Несколько текстов: маркер может быть в любом из них (описание + анкета).
	if !DetectCrisis("всё нормально", "дома угрожают") {
		t.Error("DetectCrisis should scan all texts")
	}
}

func TestNormalizeTrack(t *testing.T) {
	cases := map[string]string{
		"ОТК-X7KD-R9MF-Q3HP":   "ОТКX7KDR9MFQ3HP",
		"otk-x7kd-r9mf-q3hp":   "ОТКX7KDR9MFQ3HP", // латинская OTK -> ОТК
		" отк-X7KД-R9mf-q3hp ": "ОТКX7KДR9MFQ3HP", // пробелы и регистр
	}
	for in, want := range cases {
		if got := NormalizeTrack(in); got != want {
			t.Errorf("NormalizeTrack(%q) = %q, want %q", in, got, want)
		}
	}
	// Один и тот же номер в разном написании даёт один хеш.
	if HashTrack("ОТК-X7KD-R9MF-Q3HP") != HashTrack("otk-x7kd-r9mf-q3hp") {
		t.Error("HashTrack should be stable across notations")
	}
}

func TestGenerateTrackNumber(t *testing.T) {
	re := regexp.MustCompile(`^ОТК-[23456789ABCDEFGHJKMNPQRSTUVWXYZ]{4}-[A-Z\d]{4}-[A-Z\d]{4}$`)
	for i := 0; i < 100; i++ {
		tn, err := GenerateTrackNumber()
		if err != nil {
			t.Fatalf("GenerateTrackNumber: %v", err)
		}
		if !re.MatchString(tn) {
			t.Fatalf("track number %q does not match expected format", tn)
		}
	}
}
