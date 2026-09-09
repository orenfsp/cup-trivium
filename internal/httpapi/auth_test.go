package httpapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"

	"otklik/internal/config"
	"otklik/internal/domain"
)

func TestNewTokenUnique(t *testing.T) {
	tok1, hash1, err := newToken()
	if err != nil {
		t.Fatal(err)
	}
	tok2, hash2, _ := newToken()
	if tok1 == tok2 {
		t.Error("токены должны быть уникальны")
	}
	if len(tok1) != 64 {
		t.Errorf("токен = %d символов, want 64 hex", len(tok1))
	}
	// В БД лежит только хеш: утечка дампа не даёт доступа к сессиям.
	if hash1 == tok1 {
		t.Error("хеш не должен совпадать с токеном")
	}
	if hashToken(tok1) != hash1 {
		t.Error("hashToken должен быть детерминирован")
	}
	if hash1 == hash2 {
		t.Error("хеши разных токенов не должны совпадать")
	}
}

func TestBearerToken(t *testing.T) {
	cases := map[string]string{
		"Bearer abc":  "abc",
		"bearer abc":  "abc",
		"BEARER x":    "x",
		"Basic abc":   "",
		"Bearer":      "",
		"Short 12345": "",
		"":            "",
	}
	for h, want := range cases {
		r := httptest.NewRequest("GET", "/", nil)
		if h != "" {
			r.Header.Set("Authorization", h)
		}
		if got := bearerToken(r); got != want {
			t.Errorf("Authorization %q: bearerToken = %q, want %q", h, got, want)
		}
	}
}

func TestSessionCookies(t *testing.T) {
	s := &Server{cfg: config.Config{CookieSecure: true, SessionTTL: 90 * time.Second}}

	w := httptest.NewRecorder()
	s.setCookie(w, staffCookie, "tok")
	cs := w.Result().Cookies()
	if len(cs) != 1 {
		t.Fatalf("expected 1 cookie, got %d", len(cs))
	}
	c := cs[0]
	if c.Name != staffCookie || c.Value != "tok" {
		t.Errorf("кука = %s=%s", c.Name, c.Value)
	}
	if c.Path != "/" {
		t.Errorf("Path = %q, want /", c.Path)
	}
	if !c.HttpOnly {
		t.Error("кука сессии должна быть HttpOnly")
	}
	if !c.Secure {
		t.Error("при COOKIE_SECURE кука должна быть Secure")
	}
	if c.SameSite != http.SameSiteLaxMode {
		t.Errorf("SameSite = %v, want Lax", c.SameSite)
	}
	if c.MaxAge != 90 {
		t.Errorf("MaxAge = %d, want 90", c.MaxAge)
	}

	w2 := httptest.NewRecorder()
	s.clearCookie(w2, staffCookie)
	c2 := w2.Result().Cookies()[0]
	if c2.Value != "" || c2.MaxAge != -1 {
		t.Errorf("сброс куки: Value=%q MaxAge=%d, want \"\" и -1", c2.Value, c2.MaxAge)
	}
}

// Без куки /api/me отвечает анонимно — страница заявителя рендерится до входа.
func TestHandleMeAnonymous(t *testing.T) {
	h := New(config.Config{SessionTTL: time.Hour}, nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest("GET", "/api/me", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("код = %d, want 200", w.Code)
	}
	var out map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	if out["role"] != "anonymous" {
		t.Errorf("role = %v, want anonymous", out["role"])
	}
}

// Хелпер actorPtr: у заявителя нет user_id — события пишутся от системы.
func TestActorPtr(t *testing.T) {
	if actorPtr(domain.Principal{Role: domain.RoleApplicant}) != nil {
		t.Error("actorPtr(заявитель) должен быть nil")
	}
	if actorPtr(domain.Principal{Role: domain.RoleOperator, UserID: uuid.New()}) == nil {
		t.Error("actorPtr(сотрудник) должен быть не nil")
	}
}
