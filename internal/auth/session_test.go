package auth

import (
	"testing"
	"time"
)

func TestSessionStore_PutGet(t *testing.T) {
	s := NewSessionStore(time.Hour)
	s.Put("abc", Session{Username: "declan", Groups: []string{"family"}})
	got, ok := s.Get("abc")
	if !ok {
		t.Fatal("expected hit")
	}
	if got.Username != "declan" {
		t.Fatalf("username = %q", got.Username)
	}
}

func TestSessionStore_TTLExpiry(t *testing.T) {
	s := NewSessionStore(10 * time.Millisecond)
	s.Put("abc", Session{Username: "declan"})
	time.Sleep(20 * time.Millisecond)
	if _, ok := s.Get("abc"); ok {
		t.Fatal("expected miss after TTL")
	}
}

func TestSessionStore_Delete(t *testing.T) {
	s := NewSessionStore(time.Hour)
	s.Put("abc", Session{Username: "declan"})
	s.Delete("abc")
	if _, ok := s.Get("abc"); ok {
		t.Fatal("expected miss after delete")
	}
}

func TestSessionStore_NewIDIsUnique(t *testing.T) {
	s := NewSessionStore(time.Hour)
	seen := map[string]bool{}
	for i := 0; i < 1000; i++ {
		id := s.NewID()
		if seen[id] {
			t.Fatalf("duplicate id at iter %d: %s", i, id)
		}
		seen[id] = true
		if len(id) < 32 {
			t.Fatalf("id too short: %q (len=%d)", id, len(id))
		}
	}
}
