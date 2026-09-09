package store

import (
	"context"
	"database/sql"

	"github.com/google/uuid"

	"otklik/internal/domain"
)

func (st *Store) PublishRecommendation(ctx context.Context, appealID uuid.UUID,
	actorID *uuid.UUID, actorRole domain.Role, recommendation string) (Appeal, error) {
	return st.withAppealLock(ctx, appealID, func(ctx context.Context, tx *sql.Tx, a Appeal) error {
		if a.Status != domain.StatusInProgress {
			return domain.ErrConflict
		}
		if _, err := tx.ExecContext(ctx, `
			UPDATE appeals SET status = 'answer_ready', recommendation = $2,
				version = version + 1, updated_at = now() WHERE id = $1`,
			appealID, recommendation); err != nil {
			return err
		}
		return addEvent(ctx, tx, EventPayload{
			AppealID: appealID, ActorID: actorID, ActorRole: actorRole,
			EventType: "status", OldValue: string(a.Status),
			NewValue: string(domain.StatusAnswerReady), Reason: "recommendation_published",
		})
	})
}

// RequestTransfer — запрос передачи; уведомление оператору создаётся в той же транзакции и атомарно с запросом.
func (st *Store) RequestTransfer(ctx context.Context, appealID uuid.UUID,
	actorID *uuid.UUID, actorRole domain.Role, reason string) (Appeal, error) {
	return st.withAppealLock(ctx, appealID, func(ctx context.Context, tx *sql.Tx, a Appeal) error {
		if a.Status.Terminal() {
			return domain.ErrConflict
		}
		if _, err := tx.ExecContext(ctx, `
			UPDATE appeals SET transfer_requested = TRUE, version = version + 1,
				updated_at = now() WHERE id = $1`, appealID); err != nil {
			return err
		}
		if err := addEvent(ctx, tx, EventPayload{
			AppealID: appealID, ActorID: actorID, ActorRole: actorRole,
			EventType: "transfer_requested", NewValue: "true", Reason: reason,
		}); err != nil {
			return err
		}
		return nil
	})
}

func (st *Store) AddContributor(ctx context.Context, appealID, expertID uuid.UUID,
	actorID *uuid.UUID, actorRole domain.Role) error {
	tx, err := st.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO appeal_participants (appeal_id, expert_id, participant_role)
		VALUES ($1, $2, 'contributor')
		ON CONFLICT (appeal_id, expert_id) DO NOTHING`, appealID, expertID); err != nil {
		return err
	}
	if err := addEvent(ctx, tx, EventPayload{
		AppealID: appealID, ActorID: actorID, ActorRole: actorRole,
		EventType: "contributor_added", NewValue: expertID.String(),
	}); err != nil {
		return err
	}
	return tx.Commit()
}

func (st *Store) ApplicantResult(ctx context.Context, appealID uuid.UUID,
	helped bool, reason string) (Appeal, error) {
	set, err := st.GetSettings(ctx)
	if err != nil {
		return Appeal{}, err
	}
	return st.withAppealLock(ctx, appealID, func(ctx context.Context, tx *sql.Tx, a Appeal) error {
		if a.Status != domain.StatusAnswerReady {
			return domain.ErrConflict
		}
		// ТЗ 5.1: число возвратов ограничивает команда (по умолчанию два).
		if !helped && a.ReturnCount >= set.MaxReturns {
			return domain.ErrReturnLimitReached
		}
		to := domain.StatusReturned
		eventReason := reason
		if helped {
			to = domain.StatusCompleted
			eventReason = "helped"
		}
		if _, err := tx.ExecContext(ctx, `
			UPDATE appeals SET status = $2,
				return_count = CASE WHEN $3 THEN return_count ELSE return_count + 1 END,
				version = version + 1, updated_at = now() WHERE id = $1`,
			appealID, string(to), helped); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO feedback (appeal_id, helped) VALUES ($1, $2)
			ON CONFLICT (appeal_id) DO UPDATE SET helped = $2`, appealID, helped); err != nil {
			return err
		}
		return addEvent(ctx, tx, EventPayload{
			AppealID: appealID, ActorRole: domain.RoleApplicant,
			EventType: "status", OldValue: string(a.Status), NewValue: string(to),
			Reason: eventReason,
		})
	})
}
