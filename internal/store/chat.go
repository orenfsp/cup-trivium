package store

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"
)

// Message — публичное сообщение чата (заявитель <-> специалист от лица сервиса).
// Имя автора заявителю не раскрывается — только роль.
type Message struct {
	ID         uuid.UUID `json:"id"`
	AppealID   uuid.UUID `json:"-"`
	AuthorType string    `json:"author_type"` // applicant | expert | operator
	Text       string    `json:"text"`
	CreatedAt  time.Time `json:"created_at"`
}

func (st *Store) CreateMessage(ctx context.Context, appealID uuid.UUID,
	authorType string, authorID *uuid.UUID, text, idempotencyKey string) (Message, error) {
	var id uuid.UUID
	err := st.DB.QueryRowContext(ctx, `
		INSERT INTO messages (appeal_id, author_type, author_id, text, idempotency_key)
		VALUES ($1, $2, $3, $4, NULLIF($5, ''))
		ON CONFLICT (idempotency_key) DO UPDATE SET text = EXCLUDED.text
		RETURNING id`,
		appealID, authorType, authorID, text, idempotencyKey).Scan(&id)
	if err != nil {
		return Message{}, err
	}
	var m Message
	err = st.DB.QueryRowContext(ctx, `
		SELECT id, appeal_id, author_type, text, created_at FROM messages WHERE id = $1`, id).
		Scan(&m.ID, &m.AppealID, &m.AuthorType, &m.Text, &m.CreatedAt)
	return m, err
}


func (st *Store) ListMessages(ctx context.Context, appealID uuid.UUID) ([]Message, error) {
	rows, err := st.DB.QueryContext(ctx, `
		SELECT id, appeal_id, author_type, text, created_at
		FROM messages WHERE appeal_id = $1 ORDER BY created_at`, appealID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Message
	for rows.Next() {
		var m Message
		if err := rows.Scan(&m.ID, &m.AppealID, &m.AuthorType, &m.Text, &m.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

// InternalNote — внутренняя заметка (физически отдельная таблица).
type InternalNote struct {
	ID         uuid.UUID `json:"id"`
	AppealID   uuid.UUID `json:"-"`
	AuthorID   uuid.UUID `json:"author_id"`
	AuthorLogin string   `json:"author_login"`
	Text       string    `json:"text"`
	CreatedAt  time.Time `json:"created_at"`
}

func (st *Store) CreateNote(ctx context.Context, appealID, authorID uuid.UUID, text string) (InternalNote, error) {
	var n InternalNote
	err := st.DB.QueryRowContext(ctx, `
		INSERT INTO internal_notes (appeal_id, author_id, text)
		VALUES ($1, $2, $3)
		RETURNING id, appeal_id, author_id, text, created_at`,
		appealID, authorID, text).
		Scan(&n.ID, &n.AppealID, &n.AuthorID, &n.Text, &n.CreatedAt)
	return n, err
}

func (st *Store) ListNotes(ctx context.Context, appealID uuid.UUID) ([]InternalNote, error) {
	rows, err := st.DB.QueryContext(ctx, `
		SELECT n.id, n.appeal_id, n.author_id, u.login, n.text, n.created_at
		FROM internal_notes n JOIN users u ON u.id = n.author_id
		WHERE n.appeal_id = $1 ORDER BY n.created_at`, appealID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []InternalNote
	for rows.Next() {
		var n InternalNote
		if err := rows.Scan(&n.ID, &n.AppealID, &n.AuthorID, &n.AuthorLogin, &n.Text, &n.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, n)
	}
	return out, rows.Err()
}

// ---- Оценка ----

type Feedback struct {
	Helped  *bool   `json:"helped"`
	Rating  *int    `json:"rating"`
	Comment *string `json:"comment"`
}

func (st *Store) UpsertFeedback(ctx context.Context, appealID uuid.UUID, rating *int, comment *string) error {
	_, err := st.DB.ExecContext(ctx, `
		INSERT INTO feedback (appeal_id, rating, comment)
		VALUES ($1, $2, $3)
		ON CONFLICT (appeal_id) DO UPDATE SET rating = $2, comment = $3`,
		appealID, rating, comment)
	return err
}

// ---- Кризисный контакт (отдельно защищённые данные) ----

func (st *Store) GetCrisisContact(ctx context.Context, appealID uuid.UUID) (string, error) {
	var contact string
	err := st.DB.QueryRowContext(ctx,
		`SELECT contact FROM crisis_contacts WHERE appeal_id = $1`, appealID).Scan(&contact)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil
	}
	return contact, err
}

// ---- Жалоба на специалиста ----

func (st *Store) CreateComplaint(ctx context.Context, appealID uuid.UUID, text string) error {
	_, err := st.DB.ExecContext(ctx,
		`INSERT INTO complaints (appeal_id, text) VALUES ($1, $2)`, appealID, text)
	return err
}

type Complaint struct {
	ID        uuid.UUID `json:"id"`
	AppealID  uuid.UUID `json:"appeal_id"`
	Text      string    `json:"text"`
	CreatedAt time.Time `json:"created_at"`
}

func (st *Store) ListComplaints(ctx context.Context) ([]Complaint, error) {
	rows, err := st.DB.QueryContext(ctx,
		`SELECT id, appeal_id, text, created_at FROM complaints ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Complaint
	for rows.Next() {
		var c Complaint
		if err := rows.Scan(&c.ID, &c.AppealID, &c.Text, &c.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}
