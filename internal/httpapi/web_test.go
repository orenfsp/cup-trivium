package httpapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"otklik/internal/config"
)

func testCfg() config.Config {
	return config.Config{SessionTTL: time.Hour}
}

// ТЗ 4.2: каждая роль — отдельные страницы; HTML не кэшируется (он разный для 8080/8081),
// плейсхолдер режима обязательно подставляется.
func TestApplicantPagesServe(t *testing.T) {
	h := New(testCfg(), nil)
	for _, p := range []string{"/", "/new", "/track", "/appeal"} {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, httptest.NewRequest("GET", p, nil))
		if w.Code != http.StatusOK {
			t.Errorf("GET %s: код = %d, want 200", p, w.Code)
			continue
		}
		if ct := w.Header().Get("Content-Type"); !strings.HasPrefix(ct, "text/html") {
			t.Errorf("GET %s: Content-Type = %q", p, ct)
		}
		if cc := w.Header().Get("Cache-Control"); cc != "no-store" {
			t.Errorf("GET %s: Cache-Control = %q, want no-store", p, cc)
		}
		body := w.Body.String()
		if strings.Contains(body, "__OTKLIK_UI_MODE__") {
			t.Errorf("GET %s: плейсхолдер режима не заменён", p)
		}
		if !strings.Contains(body, "applicant") {
			t.Errorf("GET %s: в теле отсутствует режим applicant", p)
		}
	}
}

func TestStaffPagesServe(t *testing.T) {
	h := NewStaff(testCfg(), nil)
	for _, p := range []string{"/", "/login", "/operator", "/expert", "/admin", "/detail", "/detail/00000000-0000-0000-0000-000000000000"} {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, httptest.NewRequest("GET", p, nil))
		if w.Code != http.StatusOK {
			t.Errorf("GET %s: код = %d, want 200", p, w.Code)
			continue
		}
		body := w.Body.String()
		if strings.Contains(body, "__OTKLIK_UI_MODE__") {
			t.Errorf("GET %s: плейсхолдер режима не заменён", p)
		}
		if !strings.Contains(body, "staff") {
			t.Errorf("GET %s: в теле отсутствует режим staff", p)
		}
	}
}

func TestUnknownPage404(t *testing.T) {
	for _, h := range []http.Handler{New(testCfg(), nil), NewStaff(testCfg(), nil)} {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, httptest.NewRequest("GET", "/no-such-page", nil))
		if w.Code != http.StatusNotFound {
			t.Errorf("код = %d, want 404", w.Code)
		}
	}
}

func TestHealthOnBothPorts(t *testing.T) {
	for _, h := range []http.Handler{New(testCfg(), nil), NewStaff(testCfg(), nil)} {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, httptest.NewRequest("GET", "/api/health", nil))
		if w.Code != http.StatusOK {
			t.Fatalf("код = %d, want 200", w.Code)
		}
		var out map[string]string
		if err := json.Unmarshal(w.Body.Bytes(), &out); err != nil {
			t.Fatal(err)
		}
		if out["status"] != "ok" {
			t.Errorf("status = %q, want ok", out["status"])
		}
	}
}

// Публичные справочники доступны без аутентификации и не требуют БД.
func TestPublicReferenceData(t *testing.T) {
	h := New(testCfg(), nil)

	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest("GET", "/api/intake-questions", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("intake-questions: код = %d", w.Code)
	}
	var q struct {
		Questions []struct {
			Text    string   `json:"text"`
			Options []string `json:"options"`
		} `json:"questions"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &q); err != nil {
		t.Fatal(err)
	}
	if len(q.Questions) != 4 {
		t.Errorf("вопросов = %d, want 4 (ТЗ)", len(q.Questions))
	}

	w = httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest("GET", "/api/help/crisis", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("help/crisis: код = %d", w.Code)
	}
	var c struct {
		Contacts []map[string]any `json:"contacts"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &c); err != nil {
		t.Fatal(err)
	}
	if len(c.Contacts) < 3 {
		t.Errorf("кризисных контактов = %d, want >= 3", len(c.Contacts))
	}
}

func TestAssetsETagCaching(t *testing.T) {
	h := New(testCfg(), nil)

	w1 := httptest.NewRecorder()
	h.ServeHTTP(w1, httptest.NewRequest("GET", "/assets/vendor/qrcode.min.js", nil))
	if w1.Code != http.StatusOK {
		t.Fatalf("ассет: код = %d", w1.Code)
	}
	etag := w1.Header().Get("ETag")
	if etag == "" {
		t.Fatal("ETag не выставлен")
	}
	if cc := w1.Header().Get("Cache-Control"); cc != "no-cache" {
		t.Errorf("Cache-Control = %q, want no-cache", cc)
	}

	r := httptest.NewRequest("GET", "/assets/vendor/qrcode.min.js", nil)
	r.Header.Set("If-None-Match", etag)
	w2 := httptest.NewRecorder()
	h.ServeHTTP(w2, r)
	if w2.Code != http.StatusNotModified {
		t.Errorf("If-None-Match: код = %d, want 304", w2.Code)
	}

	w3 := httptest.NewRecorder()
	h.ServeHTTP(w3, httptest.NewRequest("GET", "/assets/missing.js", nil))
	if w3.Code != http.StatusNotFound {
		t.Errorf("несуществующий ассет: код = %d, want 404", w3.Code)
	}
}

// Защищённые эндпоинты без аутентификации отдают 401, а не 5xx.
func TestUnauthenticatedGuards(t *testing.T) {
	pub := New(testCfg(), nil)
	w := httptest.NewRecorder()
	pub.ServeHTTP(w, httptest.NewRequest("GET", "/api/appeals/me/", nil))
	if w.Code != http.StatusUnauthorized {
		t.Errorf("applicant guard: код = %d, want 401", w.Code)
	}

	staff := NewStaff(testCfg(), nil)
	guards := []string{
		"/api/operator/queue", "/api/admin/stats", "/api/expert/appeals",
		"/api/appeals/00000000-0000-0000-0000-000000000000/",
	}
	for _, g := range guards {
		w := httptest.NewRecorder()
		staff.ServeHTTP(w, httptest.NewRequest("GET", g, nil))
		if w.Code != http.StatusUnauthorized {
			t.Errorf("GET %s без аутентификации: код = %d, want 401", g, w.Code)
		}
	}
}

// Валидация создания обращения отсекает мусор до обращения к хранилищу,
// поэтому тест не требует БД.
func TestCreateAppealValidation(t *testing.T) {
	h := New(testCfg(), nil)
	post := func(payload string) int {
		w := httptest.NewRecorder()
		r := httptest.NewRequest("POST", "/api/appeals", strings.NewReader(payload))
		r.Header.Set("Content-Type", "application/json")
		h.ServeHTTP(w, r)
		return w.Code
	}
	if code := post(`{"applicant_type":"alien","description":"достаточно длинное описание"}`); code != http.StatusBadRequest {
		t.Errorf("unknown applicant_type: код = %d, want 400", code)
	}
	if code := post(`{"applicant_type":"schoolchild","description":"коротко"}`); code != http.StatusBadRequest {
		t.Errorf("short description: код = %d, want 400", code)
	}
	if code := post(`{"applicant_type":"schoolchild","free_text":false,"description":"достаточно длинное описание"}`); code != http.StatusBadRequest {
		t.Errorf("no category_id: код = %d, want 400", code)
	}
	if code := post(`{bad`); code != http.StatusBadRequest {
		t.Errorf("broken JSON: код = %d, want 400", code)
	}
}

func TestVerifyTrackRequiresNumber(t *testing.T) {
	h := New(testCfg(), nil)
	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/api/appeals/track", strings.NewReader(`{"track_number":"  "}`))
	r.Header.Set("Content-Type", "application/json")
	h.ServeHTTP(w, r)
	if w.Code != http.StatusBadRequest {
		t.Errorf("пустой трек-номер: код = %d, want 400", w.Code)
	}
}
