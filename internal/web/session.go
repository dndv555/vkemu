package web

import (
	"crypto/rand"
	"encoding/hex"
	"sync"
)

const cookieName = "vkemu_web"

type sessionStore struct {
	mu       sync.Mutex
	sessions map[string]int
}

func newSessionStore() *sessionStore {
	return &sessionStore{sessions: map[string]int{}}
}

func (s *sessionStore) create(uid int) string {
	token := randomToken()
	s.mu.Lock()
	s.sessions[token] = uid
	s.mu.Unlock()
	return token
}

func (s *sessionStore) uid(token string) (int, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	uid, ok := s.sessions[token]
	return uid, ok
}

func (s *sessionStore) drop(token string) {
	s.mu.Lock()
	delete(s.sessions, token)
	s.mu.Unlock()
}

func randomToken() string {
	buf := make([]byte, 24)
	if _, err := rand.Read(buf); err != nil {
		return hex.EncodeToString([]byte("fallback-token-value"))
	}
	return hex.EncodeToString(buf)
}
