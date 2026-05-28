package notes

import (
	"context"
	"database/sql"
	_ "embed"
	"errors"
	"fmt"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

//go:embed migrations/001_initial.sql
var initialSchema string

const maxActiveNotes = 50

type Note struct {
	ID        int64
	Body      string
	Author    string
	CreatedAt time.Time
}

type Repo struct {
	db *sql.DB
}

func NewRepo(dsn string) (*Repo, error) {
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}
	if err := db.Ping(); err != nil {
		return nil, err
	}
	return &Repo{db: db}, nil
}

func (r *Repo) Close() error { return r.db.Close() }

func (r *Repo) Migrate(ctx context.Context) error {
	_, err := r.db.ExecContext(ctx, initialSchema)
	return err
}

func (r *Repo) Add(ctx context.Context, body, author string) (int64, error) {
	body = strings.TrimSpace(body)
	if body == "" {
		return 0, errors.New("notes: empty body")
	}
	if author == "" {
		return 0, errors.New("notes: empty author")
	}
	now := time.Now().Unix()
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer func() { _ = tx.Rollback() }()

	res, err := tx.ExecContext(ctx,
		`INSERT INTO notes (body, author, created_at) VALUES (?, ?, ?)`,
		body, author, now)
	if err != nil {
		return 0, fmt.Errorf("notes: insert: %w", err)
	}
	id, _ := res.LastInsertId()

	// Enforce cap: soft-delete any note beyond the 50 newest.
	_, err = tx.ExecContext(ctx, `
		UPDATE notes SET deleted_at = ?
		WHERE id IN (
		  SELECT id FROM notes
		  WHERE deleted_at IS NULL
		  ORDER BY created_at DESC, id DESC
		  LIMIT -1 OFFSET ?
		)`, now, maxActiveNotes)
	if err != nil {
		return 0, fmt.Errorf("notes: cap enforce: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return id, nil
}

func (r *Repo) Delete(ctx context.Context, id int64) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE notes SET deleted_at = ? WHERE id = ? AND deleted_at IS NULL`,
		time.Now().Unix(), id)
	return err
}

func (r *Repo) ListActive(ctx context.Context, limit int) ([]Note, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, body, author, created_at FROM notes
		WHERE deleted_at IS NULL
		ORDER BY created_at DESC, id DESC
		LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var out []Note
	for rows.Next() {
		var n Note
		var ts int64
		if err := rows.Scan(&n.ID, &n.Body, &n.Author, &ts); err != nil {
			return nil, err
		}
		n.CreatedAt = time.Unix(ts, 0)
		out = append(out, n)
	}
	return out, rows.Err()
}
