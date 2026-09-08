package store

import (
	"context"
	"database/sql"

	"github.com/google/uuid"

	"otklik/internal/domain"
)

func (st *Store) AssignExpert(ctx context.Context, appealID, expertID uuid.UUID,
	actorID *uuid.UUID, actorRole domain.Role, reason string) (Appeal, error) {
	return st.withAppealLock(ctx, appealID, func(ctx context.Context, tx *sql.Tx, a Appeal) error {
		allowed := a.Status == domain.StatusNew || a.Status == domain.StatusReturned || a.TransferRequested
		if !allowed && actorRole != domain.RoleAdmin {
			return domain.ErrConflict
		}
		if a.Status.Terminal() {
			return domain.ErrConflict
		}
		// Ровно один ответственный: старая запись удаляется, новая создаётся.
		if _, err := tx.ExecContext(ctx,
			`DELETE FROM appeal_participants WHERE appeal_id = $1 AND participant_role = 'responsible'`,
			appealID); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO appeal_participants (appeal_id, expert_id, participant_role)
			VALUES ($1, $2, 'responsible')
			ON CONFLICT (appeal_id, expert_id) DO UPDATE SET participant_role = 'responsible'`,
			appealID, expertID); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `
			UPDATE appeals SET assigned_expert_id = $2, transfer_requested = FALSE,
				status = 'assigned', version = version + 1, updated_at = now()
			WHERE id = $1`, appealID, expertID); err != nil {
			return err
		}
		old := ""
		if a.AssignedExpertID != nil {
			old = a.AssignedExpertID.String()
		}
		return addEvent(ctx, tx, EventPayload{
			AppealID: appealID, ActorID: actorID, ActorRole: actorRole,
			EventType: "assign", OldValue: old, NewValue: expertID.String(), Reason: reason,
		})
	})
}

func (st *Store) Reject(ctx context.Context, appealID uuid.UUID,
	actorID *uuid.UUID, actorRole domain.Role, reason string) (Appeal, error) {
	return st.withAppealLock(ctx, appealID, func(ctx context.Context, tx *sql.Tx, a Appeal) error {
		if a.Status != domain.StatusNew && a.Status != domain.StatusReturned {
			return domain.ErrConflict
		}
		if _, err := tx.ExecContext(ctx, `
			UPDATE appeals SET status = 'rejected', rejection_reason = $2,
				version = version + 1, updated_at = now() WHERE id = $1`,
			appealID, reason); err != nil {
			return err
		}
		return addEvent(ctx, tx, EventPayload{
			AppealID: appealID, ActorID: actorID, ActorRole: actorRole,
			EventType: "status", OldValue: string(a.Status),
			NewValue: string(domain.StatusRejected), Reason: reason,
		})
	})
}

func (st *Store) CompleteByOperator(ctx context.Context, appealID uuid.UUID,
	actorID *uuid.UUID, actorRole domain.Role, recommendation string) (Appeal, error) {
	return st.withAppealLock(ctx, appealID, func(ctx context.Context, tx *sql.Tx, a Appeal) error {
		if a.Status != domain.StatusNew {
			return domain.ErrConflict
		}
		if _, err := tx.ExecContext(ctx, `
			UPDATE appeals SET status = 'completed', recommendation = $2,
				version = version + 1, updated_at = now() WHERE id = $1`,
			appealID, recommendation); err != nil {
			return err
		}
		return addEvent(ctx, tx, EventPayload{
			AppealID: appealID, ActorID: actorID, ActorRole: actorRole,
			EventType: "status", OldValue: string(a.Status),
			NewValue: string(domain.StatusCompleted), Reason: "resolved_by_operator",
		})
	})
}
