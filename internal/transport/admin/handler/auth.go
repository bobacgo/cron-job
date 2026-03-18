package handler

import (
	"crypto/rand"
	"encoding/hex"
	"os"
	"sync"
	"time"
)

const authCookieName = "cronjob_session"

type session struct {
	user      string
	expiresAt time.Time
}

type authService struct {
	mu       sync.RWMutex
	sessions map[string]session
	user     string
	pass     string
	ttl      time.Duration
}

func newAuthService() *authService {
	user := os.Getenv("ADMIN_USER")
	if user == "" {
		user = "admin"
	}
	pass := os.Getenv("ADMIN_PASSWORD")
	if pass == "" {
		pass = "admin123"
	}
	return &authService{
		sessions: make(map[string]session),
		user:     user,
		pass:     pass,
		ttl:      24 * time.Hour,
	}
}

func (a *authService) authenticate(username, password string) (string, bool) {
	if username != a.user || password != a.pass {
		return "", false
	}
	token, err := randomToken(24)
	if err != nil {
		return "", false
	}
	now := time.Now().UTC()
	a.mu.Lock()
	a.cleanupLocked(now)
	a.sessions[token] = session{user: username, expiresAt: now.Add(a.ttl)}
	a.mu.Unlock()
	return token, true
}

func (a *authService) userByToken(token string) (string, bool) {
	if token == "" {
		return "", false
	}
	now := time.Now().UTC()
	a.mu.RLock()
	s, ok := a.sessions[token]
	a.mu.RUnlock()
	if !ok || s.expiresAt.Before(now) {
		if ok {
			a.mu.Lock()
			delete(a.sessions, token)
			a.mu.Unlock()
		}
		return "", false
	}
	return s.user, true
}

func (a *authService) removeSession(token string) {
	if token == "" {
		return
	}
	a.mu.Lock()
	delete(a.sessions, token)
	a.mu.Unlock()
}

func (a *authService) cleanupLocked(now time.Time) {
	for token, s := range a.sessions {
		if s.expiresAt.Before(now) {
			delete(a.sessions, token)
		}
	}
}

func randomToken(size int) (string, error) {
	buf := make([]byte, size)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}
