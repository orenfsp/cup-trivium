package httpapi

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"otklik/internal/domain"
)

func newToken() (token, tokenHash string, err error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", "", err
	}
	token = hex.EncodeToString(buf)
	return token, hashToken(token), nil
}

func hashToken(token string) string {
	h := sha256.Sum256([]byte(token))
	return hex.EncodeToString(h[:])
}

func (s *Server) setCookie(w http.ResponseWriter, name, token string) {
	http.SetCookie(w, &http.Cookie{
		Name:     name,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		Secure:   s.cfg.CookieSecure,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   int(s.cfg.SessionTTL.Seconds()),
	})
}

func (s *Server) clearCookie(w http.ResponseWriter, name string) {
	http.SetCookie(w, &http.Cookie{
		Name: name, Value: "", Path: "/", HttpOnly: true,
		Secure: s.cfg.CookieSecure, SameSite: http.SameSiteLaxMode, MaxAge: -1,
	})
}

type loginReq struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	var req loginReq
	if !decodeJSON(w, r, &req) {
		return
	}
	req.Login = strings.TrimSpace(req.Login)
	if req.Login == "" || req.Password == "" {
		writeJSON(w, http.StatusBadRequest, errorResp{"login and password are required"})
		return
	}
	u, hash, err := s.st.GetUserByLogin(r.Context(), req.Login)
	if err == nil {
		err = bcrypt.CompareHashAndPassword([]byte(hash), []byte(req.Password))
	}
	if err != nil {
		if !s.rl.allow("login:"+clientIP(r), 10, time.Minute) {
			writeJSON(w, http.StatusTooManyRequests, errorResp{"too many attempts, try later"})
			return
		}
		writeJSON(w, http.StatusUnauthorized, errorResp{"invalid credentials"})
		return
	}
	s.rl.reset("login:" + clientIP(r))
	if !u.Active {
		writeJSON(w, http.StatusForbidden, errorResp{"user is deactivated"})
		return
	}

	token, tokenHash, err := newToken()
	if err != nil {
		writeErr(w, err)
		return
	}
	if err := s.st.CreateStaffSession(r.Context(), tokenHash, u.ID, time.Now().Add(s.cfg.SessionTTL)); err != nil {
		writeErr(w, err)
		return
	}
	s.setCookie(w, staffCookie, token)
	// Токен дублируется в ответе: фронтенд шлёт его в Authorization — разные учётки в разных вкладках.
	writeJSON(w, http.StatusOK, map[string]any{
		"user_id":          u.ID,
		"login":            u.Login,
		"role":             u.Role,
		"specialist_group": u.SpecialistGroup,
		"active":           u.Active,
		"session_token":    token,
	})
}

func bearerToken(r *http.Request) string {
	h := r.Header.Get("Authorization")
	if len(h) > 7 && strings.EqualFold(h[:7], "Bearer ") {
		return strings.TrimSpace(h[7:])
	}
	return ""
}

func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	tok := ""
	if c, err := r.Cookie(staffCookie); err == nil && c.Value != "" {
		tok = c.Value
	}
	if bt := bearerToken(r); bt != "" && tok == "" {
		tok = bt
	}
	if tok != "" {
		_ = s.st.DeleteStaffSession(r.Context(), hashToken(tok))
	}
	s.clearCookie(w, staffCookie)
	writeJSON(w, http.StatusOK, map[string]string{"status": "logged out"})
}

func (s *Server) handleApplicantLogout(w http.ResponseWriter, r *http.Request) {
	if c, err := r.Cookie(applicantCookie); err == nil && c.Value != "" {
		_ = s.st.DeleteApplicantSession(r.Context(), hashToken(c.Value))
	}
	s.clearCookie(w, applicantCookie)
	writeJSON(w, http.StatusOK, map[string]string{"status": "logged out"})
}

func (s *Server) handleMe(w http.ResponseWriter, r *http.Request) {
	p, ok := principalFrom(r.Context())
	if !ok {
		writeJSON(w, http.StatusOK, map[string]any{"role": "anonymous"})
		return
	}
	resp := map[string]any{"role": p.Role}
	if p.IsStaff() {
		resp["login"] = p.Login
		resp["user_id"] = p.UserID
	} else {
		resp["appeal_id"] = p.AppealID
	}
	writeJSON(w, http.StatusOK, resp)
}

func actorPtr(p domain.Principal) *uuid.UUID {
	if !p.IsStaff() {
		return nil
	}
	id := p.UserID
	return &id
}
