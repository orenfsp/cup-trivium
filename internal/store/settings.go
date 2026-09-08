package store

import (
	"context"
	"database/sql"
	"errors"

	"github.com/google/uuid"
)

// DefaultExpertActiveLimit — лимит активных обращений на специалиста,
// пока администратор не задал иное (ТЗ п.4.5).
const DefaultExpertActiveLimit = 5

// Settings — настройки маршрутизации, задаёт администратор (ТЗ п.4.5).
// Лимит мягкий: подсказка и предупреждение оператору, назначение — за человеком.
type Settings struct {
	ExpertActiveLimit int `json:"expert_active_limit"`
}

func (st *Store) GetSettings(ctx context.Context) (Settings, error) {
	var s Settings
	s.ExpertActiveLimit = DefaultExpertActiveLimit
	var n sql.NullInt64
	err := st.DB.QueryRowContext(ctx,
		`SELECT value FROM settings WHERE key = 'expert_active_limit'`).Scan(&n)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return s, err
	}
	if n.Valid && n.Int64 > 0 {
		s.ExpertActiveLimit = int(n.Int64)
	}
	return s, nil
}

// SetExpertActiveLimit обновляет лимит активных обращений на специалиста.
func (st *Store) SetExpertActiveLimit(ctx context.Context, limit int) error {
	_, err := st.DB.ExecContext(ctx, `
		INSERT INTO settings (key, value, updated_at)
		VALUES ('expert_active_limit', $1, now())
		ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value, updated_at = now()`,
		limit)
	return err
}

// ExpertLoad — число активных обращений (assigned, in_progress,
// needs_clarification, answer_ready) по каждому специалисту (ТЗ п.4.5).
func (st *Store) ExpertLoad(ctx context.Context) (map[uuid.UUID]int, error) {
	rows, err := st.DB.QueryContext(ctx, `
		SELECT assigned_expert_id, count(*)
		FROM appeals
		WHERE assigned_expert_id IS NOT NULL
		  AND status IN ('assigned', 'in_progress', 'needs_clarification', 'answer_ready')
		GROUP BY assigned_expert_id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make(map[uuid.UUID]int)
	for rows.Next() {
		var id uuid.UUID
		var n int
		if err := rows.Scan(&id, &n); err != nil {
			return nil, err
		}
		out[id] = n
	}
	return out, rows.Err()
}