package store

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type Category struct {
	ID              uuid.UUID `json:"id"`
	Name            string    `json:"name"`
	SpecialistGroup string    `json:"specialist_group"`
	Active          bool      `json:"active"`
	FreeForm        bool      `json:"free_form"`
	CreatedAt       time.Time `json:"created_at"`
}

func (st *Store) CreateCategory(ctx context.Context, name, group string, freeForm bool) (Category, error) {
	var c Category
	err := st.DB.QueryRowContext(ctx, `
		INSERT INTO categories (name, specialist_group, free_form)
		VALUES ($1, $2, $3)
		RETURNING id, name, specialist_group, active, free_form, created_at`,
		name, group, freeForm).
		Scan(&c.ID, &c.Name, &c.SpecialistGroup, &c.Active, &c.FreeForm, &c.CreatedAt)
	return c, err
}

func (st *Store) ListCategoriesPublic(ctx context.Context) ([]Category, error) {
	return st.listCategories(ctx, true)
}

func (st *Store) ListCategoriesAll(ctx context.Context) ([]Category, error) {
	return st.listCategories(ctx, false)
}

func (st *Store) listCategories(ctx context.Context, onlyActive bool) ([]Category, error) {
	rows, err := st.DB.QueryContext(ctx, `
		SELECT id, name, specialist_group, active, free_form, created_at
		FROM categories WHERE ($1 = FALSE OR active)
		ORDER BY created_at`, onlyActive)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Category
	for rows.Next() {
		var c Category
		if err := rows.Scan(&c.ID, &c.Name, &c.SpecialistGroup, &c.Active, &c.FreeForm, &c.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func (st *Store) GetCategory(ctx context.Context, id uuid.UUID) (Category, error) {
	row := st.DB.QueryRowContext(ctx, `
		SELECT id, name, specialist_group, active, free_form, created_at
		FROM categories WHERE id = $1`, id)
	var c Category
	err := row.Scan(&c.ID, &c.Name, &c.SpecialistGroup, &c.Active, &c.FreeForm, &c.CreatedAt)
	return c, err
}

func (st *Store) UpdateCategory(ctx context.Context, id uuid.UUID,
	name, group *string, active *bool) error {
	_, err := st.DB.ExecContext(ctx, `
		UPDATE categories SET
			name = COALESCE($2, name),
			specialist_group = COALESCE($3, specialist_group),
			active = COALESCE($4, active)
		WHERE id = $1`, id, name, group, active)
	return err
}
