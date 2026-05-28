package auth

import (
	"crypto/rand"
	"encoding/hex"
	"sync"
	"time"
)

// Session is the per-user state held server-side.
type Session struct {
	Username   string   // preferred_username from claim
	Name       string   // name claim (display)
	Email      string   // email claim
	Groups     []string // groups claim
	Expiry     time.Time
	RawIDToken string // raw OIDC id_token JWT, kept verbatim for RP-initiated logout id_token_hint
}

// SessionStore is an in-memory store keyed by opaque session ID.
// Pod restart loses all sessions — accepted v1 trade-off (see spec §15 Q4).
type SessionStore struct {
	ttl time.Duration
	mu  sync.RWMutex
	m   map[string]Session
}

// NewSessionStore returns a session store with the given TTL.
// A background goroutine evicts expired sessions every 5 minutes.
func NewSessionStore(ttl time.Duration) *SessionStore {
	s := &SessionStore{ttl: ttl, m: make(map[string]Session)}
	go s.evictLoop()
	return s
}

// NewID returns a fresh, cryptographically-random session ID (256-bit hex).
func (s *SessionStore) NewID() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		panic("crypto/rand failed: " + err.Error())
	}
	return hex.EncodeToString(b)
}

// Put stores a session, computing Expiry = now + ttl.
func (s *SessionStore) Put(id string, sess Session) {
	sess.Expiry = time.Now().Add(s.ttl)
	s.mu.Lock()
	s.m[id] = sess
	s.mu.Unlock()
}

// Get returns the session and whether it was found and unexpired.
func (s *SessionStore) Get(id string) (Session, bool) {
	s.mu.RLock()
	sess, ok := s.m[id]
	s.mu.RUnlock()
	if !ok {
		return Session{}, false
	}
	if time.Now().After(sess.Expiry) {
		s.Delete(id)
		return Session{}, false
	}
	return sess, true
}

// Delete removes a session.
func (s *SessionStore) Delete(id string) {
	s.mu.Lock()
	delete(s.m, id)
	s.mu.Unlock()
}

func (s *SessionStore) evictLoop() {
	t := time.NewTicker(5 * time.Minute)
	defer t.Stop()
	for range t.C {
		now := time.Now()
		s.mu.Lock()
		for id, sess := range s.m {
			if now.After(sess.Expiry) {
				delete(s.m, id)
			}
		}
		s.mu.Unlock()
	}
}
