package store

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type Event struct {
	ID        uuid.UUID  `json:"id"`
	ActorID   *uuid.UUID `json:"actor_id"`
	ActorRole string     `json:"actor_role"`
	EventType string     `json:"event_type"`
	OldValue  *string    `json:"old_value"`
	NewValue  *string    `json:"new_value"`
	Reason    *string    `json:"reason"`
	CreatedAt time.Time  `json:"created_at"`
}

func (st *Store) ListEvents(ctx context.Context, appealID uuid.UUID) ([]Event, error) {
	rows, err := st.DB.QueryContext(ctx, `
		SELECT id, actor_id, actor_role, event_type, old_value, new_value, reason, created_at
		FROM appeal_events WHERE appeal_id = $1 ORDER BY created_at`, appealID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Event
	for rows.Next() {
		var e Event
		if err := rows.Scan(&e.ID, &e.ActorID, &e.ActorRole, &e.EventType,
			&e.OldValue, &e.NewValue, &e.Reason, &e.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

type StatusHistoryItem struct {
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}

func (st *Store) ListStatusHistoryForApplicant(ctx context.Context, appealID uuid.UUID) ([]StatusHistoryItem, error) {
	rows, err := st.DB.QueryContext(ctx, `
		SELECT new_value, created_at FROM appeal_events
		WHERE appeal_id = $1 AND event_type = 'status' AND new_value IS NOT NULL
		ORDER BY created_at`, appealID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []StatusHistoryItem
	for rows.Next() {
		var it StatusHistoryItem
		if err := rows.Scan(&it.Status, &it.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, it)
	}
	return out, rows.Err()
}
