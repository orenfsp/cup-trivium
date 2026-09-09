package httpapi

import (
	"bytes"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"

	"otklik/internal/domain"
)

func TestDecodeJSONOK(t *testing.T) {
	var v struct {
		A int `json:"a"`
	}
	r := httptest.NewRequest("POST", "/", strings.NewReader(`{"a":42}`))
	w := httptest.NewRecorder()
	if !decodeJSON(w, r, &v) {
		t.Fatalf("decodeJSON = false, want true (body: %s)", w.Body.String())
	}
	if v.A != 42 {
		t.Errorf("v.A = %d, want 42", v.A)
	}
	if w.Code != http.StatusOK {
		t.Errorf("при успехе статус не должен перезаписываться, got %d", w.Code)
	}
}

func TestDecodeJSONInvalid(t *testing.T) {
	r := httptest.NewRequest("POST", "/", strings.NewReader(`{bad json`))
	w := httptest.NewRecorder()
	if decodeJSON(w, r, &struct{}{}) {
		t.Error("decodeJSON = true для битого JSON")
	}
	if w.Code != http.StatusBadRequest {
		t.Errorf("код = %d, want 400", w.Code)
	}
}

// Тело больше 1 МБ должно отсекаться ещё до разбора.
func TestDecodeJSONTooLarge(t *testing.T) {
	r := httptest.NewRequest("POST", "/", bytes.NewReader(bytes.Repeat([]byte{'a'}, maxBodySize+16)))
	w := httptest.NewRecorder()
	if decodeJSON(w, r, &struct{}{}) {
		t.Error("decodeJSON = true для тела больше лимита")
	}
	if w.Code != http.StatusBadRequest {
		t.Errorf("код = %d, want 400", w.Code)
	}
}

func TestParseUUID(t *testing.T) {
	if _, ok := parseUUID("  " + uuid.NewString() + "  "); !ok {
		t.Error("валидный UUID с пробелами должен приниматься")
	}
	for _, s := range []string{"", "not-a-uuid", "12345"} {
		if _, ok := parseUUID(s); ok {
			t.Errorf("parseUUID(%q) = true, want false", s)
		}
	}
}

func TestWriteErrMapping(t *testing.T) {
	cases := []struct {
		err  error
		code int
	}{
		{domain.ErrUnauthorized, http.StatusUnauthorized},
		{domain.ErrForbidden, http.StatusForbidden},
		{domain.ErrNotFound, http.StatusNotFound},
		{domain.ErrConflict, http.StatusConflict},
		{domain.ErrValidation, http.StatusUnprocessableEntity},
		{domain.ErrRateLimited, http.StatusTooManyRequests},
		{domain.ErrReturnLimitReached, http.StatusConflict},
		{errors.New("boom"), http.StatusInternalServerError},
	}
	for _, c := range cases {
		w := httptest.NewRecorder()
		writeErr(w, c.err)
		if w.Code != c.code {
			t.Errorf("writeErr(%v): код = %d, want %d", c.err, w.Code, c.code)
		}
		if !strings.Contains(w.Body.String(), `"error"`) {
			t.Errorf("writeErr(%v): тело должно содержать error: %s", c.err, w.Body.String())
		}
	}
}

// Лимит возвратов исчерпан — заявитель должен получить человеческое объяснение (ТЗ 5.1).
func TestWriteErrReturnLimitMessage(t *testing.T) {
	w := httptest.NewRecorder()
	writeErr(w, domain.ErrReturnLimitReached)
	if !strings.Contains(w.Body.String(), "лимит возвратов") {
		t.Errorf("текст для заявителя отсутствует: %s", w.Body.String())
	}
}

func TestClientIP(t *testing.T) {
	r := httptest.NewRequest("GET", "/", nil)
	r.RemoteAddr = "203.0.113.7:54321"
	if got := clientIP(r); got != "203.0.113.7" {
		t.Errorf("clientIP = %q, want 203.0.113.7", got)
	}
	r.RemoteAddr = "no-port-host"
	if got := clientIP(r); got != "no-port-host" {
		t.Errorf("clientIP без порта = %q", got)
	}
}
