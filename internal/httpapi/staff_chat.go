package httpapi

import (
	"net/http"
	"strings"

	"otklik/internal/domain"
)

// ---- Чат, заметки и события (сотрудники) ----

func (s *Server) handleStaffMessages(w http.ResponseWriter, r *http.Request) {
	p, _ := principalFrom(r.Context())
	// ТЗ п.2/п.4: переписка заявителя с экспертом доступна только эксперту.
	// Оператор видит лишь исходный текст обращения, администратор в переписке
	// не участвует и доступа к ней не имеет.
	if p.Role == domain.RoleOperator || p.Role == domain.RoleAdmin {
		writeJSON(w, http.StatusForbidden, errorResp{"chat is available to expert only"})
		return
	}
	id, ok := s.appealID(w, r)
	if !ok {
		return
	}
	if _, err := s.loadAppealWithAccess(r, id, p); err != nil {
		writeErr(w, err)
		return
	}
	msgs, err := s.st.ListMessages(r.Context(), id)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"messages": msgs})
}

func (s *Server) handleStaffPostMessage(w http.ResponseWriter, r *http.Request) {
	p, _ := principalFrom(r.Context())
	// После назначения обращение ведёт эксперт; оператор и администратор
	// (ТЗ п.4 — не участвует в переписке) в чат не пишут.
	if p.Role == domain.RoleOperator || p.Role == domain.RoleAdmin {
		writeJSON(w, http.StatusForbidden, errorResp{"chat is available to expert only"})
		return
	}
	id, ok := s.appealID(w, r)
	if !ok {
		return
	}
	a, err := s.loadAppealWithAccess(r, id, p)
	if err != nil {
		writeErr(w, err)
		return
	}
	if a.Status.Terminal() {
		writeJSON(w, http.StatusConflict, errorResp{"appeal is closed"})
		return
	}
	var req postMessageReq
	if !decodeJSON(w, r, &req) {
		return
	}
	if len(strings.TrimSpace(req.Text)) < 1 || len(req.Text) > 4000 {
		writeJSON(w, http.StatusBadRequest, errorResp{"message text length must be 1..4000"})
		return
	}
	authorType := "operator"
	if p.Role == domain.RoleExpert {
		authorType = "expert"
	}
	msg, err := s.st.CreateMessage(r.Context(), id, authorType, actorPtr(p), req.Text, "")
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, msg)
}

type noteReq struct {
	Text string `json:"text"`
}

func (s *Server) handleStaffNotes(w http.ResponseWriter, r *http.Request) {
	p, _ := principalFrom(r.Context())
	// ТЗ п.4.4: внутреннее обсуждение — заметки видны всем участникам
	// обращения и оператору (чтение), но никогда не видны заявителю.
	// Администратор работает с метаданными (ТЗ п.5) — заметки ему закрыты.
	if p.Role == domain.RoleAdmin {
		writeJSON(w, http.StatusForbidden, errorResp{"notes are not available to admin"})
		return
	}
	id, ok := s.appealID(w, r)
	if !ok {
		return
	}
	if _, err := s.loadAppealWithAccess(r, id, p); err != nil {
		writeErr(w, err)
		return
	}
	notes, err := s.st.ListNotes(r.Context(), id)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"notes": notes})
}

func (s *Server) handleStaffPostNote(w http.ResponseWriter, r *http.Request) {
	p, _ := principalFrom(r.Context())
	id, ok := s.appealID(w, r)
	if !ok {
		return
	}
	// ТЗ п.4.4: внутреннее обсуждение ведут специалисты-участники;
	// оператор заметки читает, но не пишет.
	if p.Role != domain.RoleExpert {
		writeJSON(w, http.StatusForbidden, errorResp{"notes are available to expert only"})
		return
	}
	if _, err := s.loadAppealWithAccess(r, id, p); err != nil {
		writeErr(w, err)
		return
	}
	var req noteReq
	if !decodeJSON(w, r, &req) {
		return
	}
	if len(strings.TrimSpace(req.Text)) < 1 || len(req.Text) > 4000 {
		writeJSON(w, http.StatusBadRequest, errorResp{"note text length must be 1..4000"})
		return
	}
	note, err := s.st.CreateNote(r.Context(), id, p.UserID, req.Text)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, note)
}

func (s *Server) handleStaffEvents(w http.ResponseWriter, r *http.Request) {
	p, _ := principalFrom(r.Context())
	if p.Role != domain.RoleOperator && p.Role != domain.RoleAdmin {
		writeJSON(w, http.StatusForbidden, errorResp{"events are available to operator and admin only"})
		return
	}
	id, ok := s.appealID(w, r)
	if !ok {
		return
	}
	events, err := s.st.ListEvents(r.Context(), id)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"events": events})
}
