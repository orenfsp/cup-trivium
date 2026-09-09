package store

import (
	"context"
	"database/sql"

	"github.com/google/uuid"
)

const DefaultExpertActiveLimit = 5
const DefaultMaxReturns = 2      // ТЗ 5.1: число возвратов ограничивает команда (разумно — два)
const DefaultNoResponseDays = 14 // ТЗ 5.1: заявитель не вернулся N дней — система закрывает обращение

type Settings struct {
	ExpertActiveLimit int `json:"expert_active_limit"`
	MaxReturns        int `json:"max_returns"`
	NoResponseDays    int `json:"no_response_days"`
}

func (st *Store) GetSettings(ctx context.Context) (Settings, error) {
	var s Settings
	s.ExpertActiveLimit = DefaultExpertActiveLimit
	s.MaxReturns = DefaultMaxReturns
	s.NoResponseDays = DefaultNoResponseDays
	rows, err := st.DB.QueryContext(ctx, `SELECT key, value FROM settings`)
	if err != nil {
		return s, err
	}
	defer rows.Close()
	for rows.Next() {
		var key string
		var n sql.NullInt64
		if err := rows.Scan(&key, &n); err != nil {
			return s, err
		}
		if !n.Valid || n.Int64 <= 0 {
			continue
		}
		switch key {
		case "expert_active_limit":
			s.ExpertActiveLimit = int(n.Int64)
		case "max_returns":
			s.MaxReturns = int(n.Int64)
		case "no_response_days":
			s.NoResponseDays = int(n.Int64)
		}
	}
	return s, rows.Err()
}

func (st *Store) setSetting(ctx context.Context, key string, value int) error {
	_, err := st.DB.ExecContext(ctx, `
		INSERT INTO settings (key, value, updated_at)
		VALUES ($1, $2, now())
		ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value, updated_at = now()`,
		key, value)
	return err
}

func (st *Store) SetExpertActiveLimit(ctx context.Context, limit int) error {
	return st.setSetting(ctx, "expert_active_limit", limit)
}

func (st *Store) SetMaxReturns(ctx context.Context, limit int) error {
	return st.setSetting(ctx, "max_returns", limit)
}

func (st *Store) SetNoResponseDays(ctx context.Context, days int) error {
	return st.setSetting(ctx, "no_response_days", days)
}

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
