package store

import (
	"context"
	"database/sql"

	"github.com/google/uuid"

	"otklik/internal/domain"
)

// AdminSetStatus — административная смена статуса в обход машины состояний
// (ТЗ п.4: разблокировка зависших обращений). Причина обязательна и попадает
// в журнал аудита вместе со старым и новым значением.
func (st *Store) AdminSetStatus(ctx context.Context, appealID uuid.UUID,
	actorID *uuid.UUID, actorRole domain.Role, to domain.Status, reason string) (Appeal, error) {
	return st.withAppealLock(ctx, appealID, func(ctx context.Context, tx *sql.Tx, a Appeal) error {
		if a.Status == to {
			return domain.ErrConflict
		}
		if _, err := tx.ExecContext(ctx, `
			UPDATE appeals SET status = $2, version = version + 1, updated_at = now()
			WHERE id = $1 AND version = $3`, appealID, string(to), a.Version); err != nil {
			return err
		}
		return addEvent(ctx, tx, EventPayload{
			AppealID: appealID, ActorID: actorID, ActorRole: actorRole,
			EventType: "status", OldValue: string(a.Status), NewValue: string(to), Reason: reason,
		})
	})
}

// ReturnForRework — возврат на доработку оператором (ТЗ п.5):
// answer_ready -> returned, счётчик возвратов растёт, причина попадает в аудит.
func (st *Store) ReturnForRework(ctx context.Context, appealID uuid.UUID,
	actorID *uuid.UUID, actorRole domain.Role, reason string) (Appeal, error) {
	return st.withAppealLock(ctx, appealID, func(ctx context.Context, tx *sql.Tx, a Appeal) error {
		if a.Status != domain.StatusAnswerReady {
			return domain.ErrConflict
		}
		if !domain.CanTransition(a.Status, domain.StatusReturned) {
			return domain.ErrValidation
		}
		res, err := tx.ExecContext(ctx, `
			UPDATE appeals SET status = $2, return_count = return_count + 1,
				version = version + 1, updated_at = now()
			WHERE id = $1 AND status = $3 AND version = $4`,
			appealID, string(domain.StatusReturned), string(domain.StatusAnswerReady), a.Version)
		if err != nil {
			return err
		}
		if n, _ := res.RowsAffected(); n == 0 {
			return domain.ErrConflict
		}
		return addEvent(ctx, tx, EventPayload{
			AppealID: appealID, ActorID: actorID, ActorRole: actorRole,
			EventType: "status", OldValue: string(a.Status),
			NewValue: string(domain.StatusReturned), Reason: reason,
		})
	})
}

// MyStats — персональная аналитика сотрудника «по себе» (ТЗ п.5).
// Оператору: назначено/в работе/завершено/отклонено по его действиям.
// Эксперту: активные и завершённые из назначенных ему, число рекомендаций,
// среднее время решения его обращений.
type MyStats struct {
	Role             string   `json:"role"`
	Assigned         int      `json:"assigned"`
	Active           int      `json:"active"`
	Resolved         int      `json:"resolved"`
	Rejected         int      `json:"rejected"`
	Recommendations  int      `json:"recommendations"`
	AvgResolutionHrs *float64 `json:"avg_resolution_hours"`
}

const activeStatusFilter = `status NOT IN ('completed','rejected','closed_no_response')`

func (st *Store) GetMyStats(ctx context.Context, role domain.Role, userID uuid.UUID) (MyStats, error) {
	var s MyStats
	s.Role = string(role)

	if role == domain.RoleOperator {
		// Обращения, которые оператор назначал (его действие «assign» в аудите).
		if err := st.DB.QueryRowContext(ctx, `
			SELECT count(DISTINCT e.appeal_id) FROM appeal_events e
			WHERE e.actor_id = $1 AND e.event_type = 'assign'`, userID).Scan(&s.Assigned); err != nil {
			return s, err
		}
		if err := st.DB.QueryRowContext(ctx, `
			SELECT count(DISTINCT e.appeal_id)
			FROM appeal_events e JOIN appeals a ON a.id = e.appeal_id
			WHERE e.actor_id = $1 AND e.event_type = 'assign' AND a.`+activeStatusFilter, userID).Scan(&s.Active); err != nil {
			return s, err
		}
		if err := st.DB.QueryRowContext(ctx, `
			SELECT count(DISTINCT e.appeal_id)
			FROM appeal_events e JOIN appeals a ON a.id = e.appeal_id
			WHERE e.actor_id = $1 AND e.event_type = 'assign' AND a.status = 'completed'`, userID).Scan(&s.Resolved); err != nil {
			return s, err
		}
		if err := st.DB.QueryRowContext(ctx, `
			SELECT count(*) FROM appeal_events
			WHERE actor_id = $1 AND event_type = 'status' AND new_value = 'rejected'`, userID).Scan(&s.Rejected); err != nil {
			return s, err
		}
		return s, nil
	}

	// Эксперт: обращения, где он участник (ответственный или соисполнитель).
	if err := st.DB.QueryRowContext(ctx, `
		SELECT count(*) FROM appeals a JOIN appeal_participants p ON p.appeal_id = a.id
		WHERE p.expert_id = $1 AND a.`+activeStatusFilter, userID).Scan(&s.Active); err != nil {
		return s, err
	}
	if err := st.DB.QueryRowContext(ctx, `
		SELECT count(*) FROM appeals a JOIN appeal_participants p ON p.appeal_id = a.id
		WHERE p.expert_id = $1 AND a.status = 'completed'`, userID).Scan(&s.Resolved); err != nil {
		return s, err
	}
	// Публикация рекомендации фиксируется переходом в answer_ready.
	if err := st.DB.QueryRowContext(ctx, `
		SELECT count(*) FROM appeal_events
		WHERE actor_id = $1 AND event_type = 'status' AND new_value = 'answer_ready'`, userID).Scan(&s.Recommendations); err != nil {
		return s, err
	}
	var avg sql.NullFloat64
	if err := st.DB.QueryRowContext(ctx, `
		SELECT avg(extract(epoch from (a.updated_at - a.created_at)) / 3600.0)
		FROM appeals a JOIN appeal_participants p ON p.appeal_id = a.id
		WHERE p.expert_id = $1 AND a.status = 'completed'`, userID).Scan(&avg); err != nil {
		return s, err
	}
	if avg.Valid {
		v := avg.Float64
		s.AvgResolutionHrs = &v
	}
	return s, nil
}

// StatRow — строка агрегата для панели аналитики.
type StatRow struct {
	Label string `json:"label"`
	Count int    `json:"count"`
}

// AdminStats — сводка для аналитики администратора (ТЗ п.4).
type AdminStats struct {
	Total             int       `json:"total"`
	Active            int       `json:"active"`
	UrgentActive      int       `json:"urgent_active"`
	Last7Days         int       `json:"last_7_days"`
	Last30Days        int       `json:"last_30_days"`
	Resolved          int       `json:"resolved"`
	AvgResolutionHrs  *float64  `json:"avg_resolution_hours"`
	ByStatus          []StatRow `json:"by_status"`
	TopCategories     []StatRow `json:"top_categories"`
	BySpecialistGroup []StatRow `json:"by_specialist_group"`
}

// AdminStats собирает агрегаты по всем обращениям: объёмы, нагрузка по
// статусам/категориям/специальностям и среднее время решения.
func (st *Store) GetAdminStats(ctx context.Context) (AdminStats, error) {
	var s AdminStats
	if err := st.DB.QueryRowContext(ctx,
		`SELECT count(*) FROM appeals`).Scan(&s.Total); err != nil {
		return s, err
	}
	if err := st.DB.QueryRowContext(ctx, `
		SELECT count(*) FROM appeals
		WHERE status NOT IN ('completed','rejected','closed_no_response')`).Scan(&s.Active); err != nil {
		return s, err
	}
	if err := st.DB.QueryRowContext(ctx, `
		SELECT count(*) FROM appeals
		WHERE priority = 'urgent'
		  AND status NOT IN ('completed','rejected','closed_no_response')`).Scan(&s.UrgentActive); err != nil {
		return s, err
	}
	if err := st.DB.QueryRowContext(ctx,
		`SELECT count(*) FROM appeals WHERE created_at > now() - interval '7 days'`).Scan(&s.Last7Days); err != nil {
		return s, err
	}
	if err := st.DB.QueryRowContext(ctx,
		`SELECT count(*) FROM appeals WHERE created_at > now() - interval '30 days'`).Scan(&s.Last30Days); err != nil {
		return s, err
	}
	// Среднее время решения: от создания до завершения (часы).
	var avg sql.NullFloat64
	if err := st.DB.QueryRowContext(ctx, `
		SELECT count(*), avg(extract(epoch from (updated_at - created_at)) / 3600.0)
		FROM appeals WHERE status = 'completed'`).Scan(&s.Resolved, &avg); err != nil {
		return s, err
	}
	if avg.Valid {
		v := avg.Float64
		s.AvgResolutionHrs = &v
	}

	rows, err := st.DB.QueryContext(ctx,
		`SELECT status, count(*) FROM appeals GROUP BY status ORDER BY count(*) DESC`)
	if err != nil {
		return s, err
	}
	defer rows.Close()
	for rows.Next() {
		var r StatRow
		if err := rows.Scan(&r.Label, &r.Count); err != nil {
			return s, err
		}
		s.ByStatus = append(s.ByStatus, r)
	}
	if err = rows.Err(); err != nil {
		return s, err
	}

	rows2, err := st.DB.QueryContext(ctx, `
		SELECT COALESCE(c.name, 'Свободная форма'), count(*) AS n
		FROM appeals a LEFT JOIN categories c ON c.id = a.category_id
		GROUP BY 1 ORDER BY n DESC LIMIT 5`)
	if err != nil {
		return s, err
	}
	defer rows2.Close()
	for rows2.Next() {
		var r StatRow
		if err := rows2.Scan(&r.Label, &r.Count); err != nil {
			return s, err
		}
		s.TopCategories = append(s.TopCategories, r)
	}
	if err = rows2.Err(); err != nil {
		return s, err
	}

	rows3, err := st.DB.QueryContext(ctx, `
		SELECT COALESCE(c.specialist_group, 'free'), count(*) AS n
		FROM appeals a LEFT JOIN categories c ON c.id = a.category_id
		GROUP BY 1 ORDER BY n DESC`)
	if err != nil {
		return s, err
	}
	defer rows3.Close()
	for rows3.Next() {
		var r StatRow
		if err := rows3.Scan(&r.Label, &r.Count); err != nil {
			return s, err
		}
		s.BySpecialistGroup = append(s.BySpecialistGroup, r)
	}
	return s, rows3.Err()
}
