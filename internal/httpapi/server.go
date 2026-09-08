// Package httpapi — HTTP-слой: маршруты chi, аутентификация по сессионным
// кукам и проекции данных для заявителей и сотрудников.
package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"

	"otklik/internal/config"
	"otklik/internal/domain"
	"otklik/internal/store"
)

const (
	staffCookie     = "otklik_staff"
	applicantCookie = "otklik_applicant"
	maxBodySize     = 1 << 20 // 1 МБ для JSON-запросов
)

// Server хранит конфигурацию, стор и простой in-memory rate limiter.
type Server struct {
	cfg      config.Config
	st       *store.Store
	rl       *rateLimiter
	presence *presenceHub
}

// New собирает маршрутизатор со всеми обработчиками.
func baseRouter() *chi.Mux {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)
	return r
}

func healthHandler(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// New собирает маршрутизатор публичной части (порт заявителей):
// создание обращений, вход по трек-номеру и сессия заявителя.
// Маршруты сотрудников на этом порту недоступны в принципе.
func New(cfg config.Config, st *store.Store) http.Handler {
	s := &Server{cfg: cfg, st: st, rl: newRateLimiter(), presence: newPresenceHub()}

	r := baseRouter()
	r.Use(s.authMW(false, true)) // только сессии заявителей

	// ---- Публичные маршруты ----
	r.Get("/api/health", healthHandler)
	r.Get("/", s.indexFor("applicant"))
	r.Handle("/assets/*", assetsHandler())
	r.Get("/api/categories", s.handlePublicCategories)
	r.Get("/api/intake-questions", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string][]domain.IntakeQuestion{"questions": domain.IntakeQuestions})
	})
	r.Get("/api/help/crisis", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string][]domain.CrisisHelp{"contacts": domain.CrisisHelpContacts})
	})

	// ---- Анонимный заявитель ----
	r.Post("/api/appeals", s.handleCreateAppeal)
	r.Post("/api/appeals/track", s.handleVerifyTrack)
	r.Get("/api/me", s.handleMe)

	// ---- Сессия заявителя (по трек-номеру) ----
	r.Group(func(r chi.Router) {
		r.Use(s.requireApplicant)
		r.Route("/api/appeals/me", func(r chi.Router) {
			r.Get("/", s.handleApplicantView)
			r.Get("/messages", s.handleApplicantMessages)
			r.Post("/messages", s.handleApplicantPostMessage)
			r.Post("/append", s.handleApplicantAppend)
			r.Post("/result", s.handleApplicantResult)
			r.Post("/feedback", s.handleApplicantFeedback)
			r.Post("/complaint", s.handleApplicantComplaint)
			r.Post("/attachments", s.handleUploadAttachment)
			r.Get("/attachments/{attachmentID}", s.handleDownloadAttachment)
			r.Post("/session/end", s.handleApplicantLogout)
		})
	})

	return r
}

// NewStaff собирает маршрутизатор служебной части (порт сотрудников):
// аутентификация, очереди, панели оператора/эксперта/админа.
// Маршруты заявителей на этом порту недоступны.
func NewStaff(cfg config.Config, st *store.Store) http.Handler {
	s := &Server{cfg: cfg, st: st, rl: newRateLimiter(), presence: newPresenceHub()}

	r := baseRouter()
	r.Use(s.authMW(true, false)) // только сессии сотрудников (токен вкладки или кука)

	// ---- Публичные маршруты ----
	r.Get("/api/health", healthHandler)
	r.Get("/", s.indexFor("staff"))
	r.Handle("/assets/*", assetsHandler())
	r.Get("/api/categories", s.handlePublicCategories) // нужно панели оператора

	// ---- Аутентификация сотрудников ----
	r.Post("/api/auth/login", s.handleLogin)
	r.Post("/api/auth/logout", s.handleLogout)
	r.Get("/api/me", s.handleMe)

	// ---- Общее для сотрудников: список активных экспертов ----
	// Нужен и оператору (назначение эксперта), и эксперту (подключение
	// коллеги-соисполнителя, ТЗ п.3), и админу. Регистрировать этот путь
	// внутри нескольких групп нельзя: в chi поздняя регистрация перекрывает
	// раннюю, и оператор получает «insufficient permissions».
	r.Group(func(r chi.Router) {
		r.Use(s.requireRole(domain.RoleExpert, domain.RoleOperator, domain.RoleAdmin))
		r.Get("/api/staff/experts", s.handleListExperts)
		// ТЗ п.5: выгрузки — оператор и админ по всем обращениям, эксперт по себе.
		r.Get("/api/export/appeals", s.handleExportAppeals)
	})

	// ---- Персональная аналитика (ТЗ п.5: «по себе») ----
	r.Group(func(r chi.Router) {
		r.Use(s.requireRole(domain.RoleOperator, domain.RoleExpert))
		r.Get("/api/mystats", s.handleMyStats)
	})

	// ---- Оператор ----
	r.Group(func(r chi.Router) {
		r.Use(s.requireRole(domain.RoleOperator, domain.RoleAdmin))
		r.Get("/api/operator/queue", s.handleOperatorQueue)
		r.Get("/api/operator/appeals", s.handleOperatorAppeals)
	})

	// ---- Эксперт ----
	r.Group(func(r chi.Router) {
		r.Use(s.requireRole(domain.RoleExpert, domain.RoleAdmin))
		r.Get("/api/expert/appeals", s.handleExpertAppeals)
	})

	// ---- Администратор ----
	r.Group(func(r chi.Router) {
		r.Use(s.requireRole(domain.RoleAdmin))
		r.Get("/api/admin/appeals", s.handleAdminAppeals)
		r.Get("/api/admin/settings", s.handleAdminGetSettings)
		r.Put("/api/admin/settings", s.handleAdminUpdateSettings)
		r.Get("/api/admin/users", s.handleAdminListUsers)
		r.Post("/api/admin/users", s.handleAdminCreateUser)
		r.Patch("/api/admin/users/{userID}", s.handleAdminPatchUser)
		r.Get("/api/admin/categories", s.handleAdminListCategories)
		r.Post("/api/admin/categories", s.handleAdminCreateCategory)
		r.Patch("/api/admin/categories/{categoryID}", s.handleAdminPatchCategory)
		r.Get("/api/admin/complaints", s.handleAdminComplaints)
		r.Get("/api/admin/stats", s.handleAdminStats)
	})
	// ---- Обращение (сотрудники) ----
	r.Route("/api/appeals/{appealID}", func(r chi.Router) {
		r.Use(s.requireStaff)
		r.Get("/", s.handleStaffGetAppeal)
		r.Get("/messages", s.handleStaffMessages)
		r.Post("/messages", s.handleStaffPostMessage)
		r.Get("/notes", s.handleStaffNotes)
		r.Post("/notes", s.handleStaffPostNote)
		r.Get("/events", s.handleStaffEvents)
		// ТЗ п.4.4 «Одновременная работа»: кто сейчас в карточке
		// обращения и кто вводит ответ (защита от двойного ответа).
		r.Get("/presence", s.handlePresenceGet)
		r.Post("/presence", s.handlePresencePost)
		r.Post("/attachments", s.handleUploadAttachment)
		r.Get("/attachments/{attachmentID}", s.handleDownloadAttachment)
		r.Post("/assign", s.handleAssign)
		r.Post("/reject", s.handleReject)
		r.Post("/complete", s.handleCompleteByOperator)
		r.Post("/priority", s.handleSetPriority)
		r.Post("/category", s.handleSetCategory)
		r.Post("/take", s.handleTakeInProgress)
		r.Post("/clarify", s.handleClarify)
		r.Post("/recommendation", s.handlePublishRecommendation)
		r.Post("/transfer-request", s.handleRequestTransfer)
		r.Post("/close-no-response", s.handleCloseNoResponse)
		r.Post("/return", s.handleReturnForRework)
		r.Post("/contributors", s.handleAddContributor)
		r.Post("/admin-status", s.handleAdminSetStatus)
	})

	return r
}

// ---- Principal (субъект доступа) ----

type principalCtxKey struct{}

// authMW восстанавливает субъект доступа. Публичный порт (заявители) слушает
// только куки заявителей, служебный — только сессии сотрудников: токен вкладки
// (Authorization: Bearer) или куку сотрудника.
func (s *Server) authMW(allowStaff, allowApplicant bool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var p domain.Principal
			found := false

			if allowStaff {
				if bt := bearerToken(r); bt != "" {
					hash := hashToken(bt)
					userID, ok, err := s.st.GetStaffSession(r.Context(), hash)
					if err == nil && ok {
						if u, err := s.st.GetUserByID(r.Context(), userID); err == nil && u.Active {
							p = domain.Principal{Role: domain.Role(u.Role), UserID: u.ID, Login: u.Login}
							found = true
						}
					}
				}
				if !found {
					if c, err := r.Cookie(staffCookie); err == nil && c.Value != "" {
						hash := hashToken(c.Value)
						userID, ok, err := s.st.GetStaffSession(r.Context(), hash)
						if err == nil && ok {
							if u, err := s.st.GetUserByID(r.Context(), userID); err == nil && u.Active {
								p = domain.Principal{Role: domain.Role(u.Role), UserID: u.ID, Login: u.Login}
								found = true
							}
						}
					}
				}
			}
			if !found && allowApplicant {
				if c, err := r.Cookie(applicantCookie); err == nil && c.Value != "" {
					hash := hashToken(c.Value)
					appealID, ok, err := s.st.GetApplicantSession(r.Context(), hash)
					if err == nil && ok {
						p = domain.Principal{Role: domain.RoleApplicant, AppealID: appealID}
						found = true
					}
				}
			}

			if found {
				r = r.WithContext(context.WithValue(r.Context(), principalCtxKey{}, p))
			}
			next.ServeHTTP(w, r)
		})
	}
}

func principalFrom(ctx context.Context) (domain.Principal, bool) {
	p, ok := ctx.Value(principalCtxKey{}).(domain.Principal)
	return p, ok
}

func (s *Server) requireApplicant(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if p, ok := principalFrom(r.Context()); ok && p.Role == domain.RoleApplicant {
			next.ServeHTTP(w, r)
			return
		}
		writeJSON(w, http.StatusUnauthorized, errorResp{"track number verification required"})
	})
}

func (s *Server) requireStaff(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if p, ok := principalFrom(r.Context()); ok && p.IsStaff() {
			next.ServeHTTP(w, r)
			return
		}
		writeJSON(w, http.StatusUnauthorized, errorResp{"staff authentication required"})
	})
}

func (s *Server) requireRole(roles ...domain.Role) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			p, ok := principalFrom(r.Context())
			if !ok || !p.IsStaff() {
				writeJSON(w, http.StatusUnauthorized, errorResp{"staff authentication required"})
				return
			}
			for _, role := range roles {
				if p.Role == role {
					next.ServeHTTP(w, r)
					return
				}
			}
			writeJSON(w, http.StatusForbidden, errorResp{"insufficient permissions"})
		})
	}
}

// ---- Простые in-memory rate limits ----

type rateLimiter struct {
	mu   sync.Mutex
	hits map[string][]time.Time
}

func newRateLimiter() *rateLimiter {
	return &rateLimiter{hits: map[string][]time.Time{}}
}

func (rl *rateLimiter) allow(key string, n int, window time.Duration) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	now := time.Now()
	kept := rl.hits[key][:0]
	for _, t := range rl.hits[key] {
		if now.Sub(t) < window {
			kept = append(kept, t)
		}
	}
	// Истёкшие ключи удаляем целиком: одноразовые посетители не должны
	// бесконечно копиться в памяти.
	if len(kept) == 0 {
		delete(rl.hits, key)
	}
	if len(kept) >= n {
		rl.hits[key] = kept
		return false
	}
	rl.hits[key] = append(kept, now)
	return true
}

// reset очищает счётчик ключа. Вызывается после успешной аутентификации:
// тот, кто предъявил верный секрет, — легитимный владелец, и прошлые
// опечатки не должны держать его в блокировке.
func (rl *rateLimiter) reset(key string) {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	delete(rl.hits, key)
}

func clientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

// ---- Вспомогательные функции ----

type errorResp struct {
	Error string `json:"error"`
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// writeErr отображает доменные ошибки в HTTP-коды.
func writeErr(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, domain.ErrUnauthorized):
		writeJSON(w, http.StatusUnauthorized, errorResp{err.Error()})
	case errors.Is(err, domain.ErrForbidden):
		writeJSON(w, http.StatusForbidden, errorResp{err.Error()})
	case errors.Is(err, domain.ErrNotFound):
		writeJSON(w, http.StatusNotFound, errorResp{"not found"})
	case errors.Is(err, domain.ErrConflict):
		writeJSON(w, http.StatusConflict, errorResp{"state conflict"})
	case errors.Is(err, domain.ErrValidation):
		writeJSON(w, http.StatusUnprocessableEntity, errorResp{"validation error"})
	case errors.Is(err, domain.ErrRateLimited):
		writeJSON(w, http.StatusTooManyRequests, errorResp{"rate limited"})
	default:
		log.Printf("httpapi: internal error: %v", err)
		writeJSON(w, http.StatusInternalServerError, errorResp{"internal error"})
	}
}

func decodeJSON(w http.ResponseWriter, r *http.Request, v any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, maxBodySize)
	if err := json.NewDecoder(r.Body).Decode(v); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResp{"invalid JSON body"})
		return false
	}
	return true
}

func parseUUID(s string) (uuid.UUID, bool) {
	id, err := uuid.Parse(strings.TrimSpace(s))
	return id, err == nil
}

