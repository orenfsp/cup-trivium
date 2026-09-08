package httpapi

import (
	"bytes"
	"net/http"
	"strconv"
	"strings"

	"otklik/internal/domain"
)

// ---- Администратор: разблокировка зависших обращений и аналитика (ТЗ п.4) ----

type adminStatusReq struct {
	Status string `json:"status"`
	Reason string `json:"reason"`
}

// handleAdminSetStatus — смена статуса любого обращения администратором.
// Причина обязательна; действие фиксируется в журнале аудита. Машина
// состояний сознательно не применяется — смысл действия в разблокировке.
func (s *Server) handleAdminSetStatus(w http.ResponseWriter, r *http.Request) {
	p, _ := principalFrom(r.Context())
	if p.Role != domain.RoleAdmin {
		writeJSON(w, http.StatusForbidden, errorResp{"status override is available to admin only"})
		return
	}
	id, ok := s.appealID(w, r)
	if !ok {
		return
	}
	var req adminStatusReq
	if !decodeJSON(w, r, &req) {
		return
	}
	to := domain.Status(strings.TrimSpace(req.Status))
	if !to.Valid() {
		writeJSON(w, http.StatusBadRequest, errorResp{"status is invalid"})
		return
	}
	req.Reason = strings.TrimSpace(req.Reason)
	if len(req.Reason) < 5 {
		writeJSON(w, http.StatusBadRequest, errorResp{"reason is required (min 5 characters)"})
		return
	}
	a, err := s.st.AdminSetStatus(r.Context(), id, actorPtr(p), p.Role, to, req.Reason)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, s.staffAppeal(r, a, p))
}

// handleAdminStats — агрегаты для панели аналитики администратора.
func (s *Server) handleAdminStats(w http.ResponseWriter, r *http.Request) {
	stats, err := s.st.GetAdminStats(r.Context())
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, stats)
}

// ---- ТЗ п.5 (матрица прав): аналитика и выгрузки ----
// Оператор и эксперт видят статистику «по себе», администратор — по всем.

// handleMyStats — персональная аналитика оператора и эксперта.
func (s *Server) handleMyStats(w http.ResponseWriter, r *http.Request) {
	p, _ := principalFrom(r.Context())
	if p.Role != domain.RoleOperator && p.Role != domain.RoleExpert {
		writeJSON(w, http.StatusForbidden, errorResp{"personal stats are available to operator and expert only"})
		return
	}
	stats, err := s.st.GetMyStats(r.Context(), p.Role, p.UserID)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, stats)
}

// handleExportAppeals — выгрузка списка обращений в CSV (без текста обращений
// и персональных данных). Оператор и администратор выгружают все обращения,
// эксперт — только назначенные ему (матрица прав: «по себе»).
func (s *Server) handleExportAppeals(w http.ResponseWriter, r *http.Request) {
	p, _ := principalFrom(r.Context())

	var buf bytes.Buffer
	buf.WriteString("\ufeff") // BOM: Excel корректно открывает UTF-8
	buf.WriteString("id,applicant_type,category,status,priority,crisis,assigned_expert,returns,created_at,updated_at\r\n")

	row := func(cols ...string) {
		for i, c := range cols {
			if strings.ContainsAny(c, ",\"\r\n") {
				c = `"` + strings.ReplaceAll(c, `"`, `""`) + `"`
			}
			if i > 0 {
				buf.WriteByte(',')
			}
			buf.WriteString(c)
		}
		buf.WriteString("\r\n")
	}

	if p.Role == domain.RoleExpert {
		items, err := s.st.ListExpertAppeals(r.Context(), p.UserID, "", "", "")
		if err != nil {
			writeErr(w, err)
			return
		}
		for _, it := range items {
			row(it.ID.String(), it.ApplicantType, deref(it.CategoryName),
				it.Status, it.Priority, boolStr(it.CrisisDetected), "",
				"0", it.CreatedAt.Format("2006-01-02 15:04"), it.UpdatedAt.Format("2006-01-02 15:04"))
		}
	} else {
		items, err := s.st.ListAppealsMeta(r.Context())
		if err != nil {
			writeErr(w, err)
			return
		}
		for _, it := range items {
			row(it.ID.String(), it.ApplicantType, deref(it.CategoryName),
				it.Status, it.Priority, boolStr(it.CrisisDetected), deref(it.ExpertLogin),
				strconv.Itoa(it.ReturnCount), it.CreatedAt.Format("2006-01-02 15:04"), it.UpdatedAt.Format("2006-01-02 15:04"))
		}
	}

	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="appeals.csv"`)
	_, _ = w.Write(buf.Bytes())
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func boolStr(b bool) string {
	if b {
		return "yes"
	}
	return "no"
}
