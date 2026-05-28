package notes

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"
)

func newTestRepo(t *testing.T) *Repo {
	t.Helper()
	r, err := NewRepo(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = r.Close() })
	if err := r.Migrate(context.Background()); err != nil {
		t.Fatal(err)
	}
	return r
}

func TestMigrate_IsIdempotent(t *testing.T) {
	r := newTestRepo(t)
	if err := r.Migrate(context.Background()); err != nil {
		t.Fatalf("second migrate must not error (pod restart safety): %v", err)
	}
}

func TestAddAndList(t *testing.T) {
	r := newTestRepo(t)
	ctx := context.Background()
	if _, err := r.Add(ctx, "Milk please", "cam"); err != nil {
		t.Fatal(err)
	}
	if _, err := r.Add(ctx, "Soccer Wed 4pm", "caitlin"); err != nil {
		t.Fatal(err)
	}
	notes, err := r.ListActive(ctx, 20)
	if err != nil {
		t.Fatal(err)
	}
	if len(notes) != 2 {
		t.Fatalf("count = %d", len(notes))
	}
	// newest first
	if notes[0].Author != "caitlin" {
		t.Fatalf("ordering wrong: %+v", notes)
	}
}

func TestSoftDelete(t *testing.T) {
	r := newTestRepo(t)
	ctx := context.Background()
	id, _ := r.Add(ctx, "x", "declan")
	if err := r.Delete(ctx, id); err != nil {
		t.Fatal(err)
	}
	notes, _ := r.ListActive(ctx, 20)
	if len(notes) != 0 {
		t.Fatalf("expected empty after delete; got %+v", notes)
	}
}

func TestAddTrimsAndRejectsEmpty(t *testing.T) {
	r := newTestRepo(t)
	if _, err := r.Add(context.Background(), "   ", "x"); err == nil {
		t.Fatal("expected error for empty body")
	}
	if _, err := r.Add(context.Background(), "", "x"); err == nil {
		t.Fatal("expected error for empty body")
	}
}

func TestGetAuthor(t *testing.T) {
	r := newTestRepo(t)
	ctx := context.Background()
	id, _ := r.Add(ctx, "x", "alice")
	got, err := r.GetAuthor(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	if got != "alice" {
		t.Fatalf("author = %q, want alice", got)
	}
	// Non-existent id returns sql.ErrNoRows.
	if _, err := r.GetAuthor(ctx, 9999); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("want sql.ErrNoRows, got %v", err)
	}
	// Soft-deleted row also returns ErrNoRows.
	_ = r.Delete(ctx, id)
	if _, err := r.GetAuthor(ctx, id); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("after soft-delete: want sql.ErrNoRows, got %v", err)
	}
}

func TestCap50SoftDeletesOldest(t *testing.T) {
	r := newTestRepo(t)
	ctx := context.Background()
	// Add 55 notes; after the 51st, the oldest should be soft-deleted.
	for i := 0; i < 55; i++ {
		if _, err := r.Add(ctx, "n", "u"); err != nil {
			t.Fatal(err)
		}
		// guarantee timestamp ordering even on fast machines
		time.Sleep(time.Millisecond)
	}
	notes, _ := r.ListActive(ctx, 100)
	if len(notes) != 50 {
		t.Fatalf("active count = %d, want 50", len(notes))
	}
}
