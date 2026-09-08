package store

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type Participant struct {
	ExpertID        uuid.UUID `json:"expert_id"`
	ExpertLogin     string    `json:"expert_login"`
	ParticipantRole string    `json:"participant_role"`
	CreatedAt       time.Time `json:"created_at"`
}

func (st *Store) IsParticipant(ctx context.Context, appealID, expertID uuid.UUID) (bool, error) {
	var ok bool
	err := st.DB.QueryRowContext(ctx, `
		SELECT EXISTS (SELECT 1 FROM appeal_participants
			WHERE appeal_id = $1 AND expert_id = $2)`, appealID, expertID).Scan(&ok)
	return ok, err
}

func (st *Store) IsResponsible(ctx context.Context, appealID, expertID uuid.UUID) (bool, error) {
	var ok bool
	err := st.DB.QueryRowContext(ctx, `
		SELECT EXISTS (SELECT 1 FROM appeal_participants
			WHERE appeal_id = $1 AND expert_id = $2 AND participant_role = 'responsible')`,
		appealID, expertID).Scan(&ok)
	return ok, err
}

func (st *Store) ListParticipants(ctx context.Context, appealID uuid.UUID) ([]Participant, error) {
	rows, err := st.DB.QueryContext(ctx, `
		SELECT p.expert_id, u.login, p.participant_role, p.created_at
		FROM appeal_participants p JOIN users u ON u.id = p.expert_id
		WHERE p.appeal_id = $1 ORDER BY p.created_at`, appealID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Participant
	for rows.Next() {
		var p Participant
		if err := rows.Scan(&p.ExpertID, &p.ExpertLogin, &p.ParticipantRole, &p.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}
