package api

import (
	"strconv"
	"strings"
	"sync"
	"time"

	"vkemu/internal/model"
	"vkemu/internal/params"
)

type tokenStore struct {
	mu     sync.Mutex
	tokens map[string]tokenEntry
}

type tokenEntry struct {
	token   string
	created int64
}

func newTokenStore() *tokenStore {
	return &tokenStore{tokens: map[string]tokenEntry{}}
}

func (t *tokenStore) issue(nonce string) string {
	token := params.MD5("token:" + nonce + ":" + strconv.FormatInt(time.Now().UnixNano(), 10))
	t.mu.Lock()
	t.tokens[nonce] = tokenEntry{token: token, created: time.Now().Unix()}
	t.mu.Unlock()
	return token
}

func (t *tokenStore) get(nonce string) (string, bool) {
	t.mu.Lock()
	defer t.mu.Unlock()
	entry, ok := t.tokens[nonce]
	if !ok {
		return "", false
	}
	if time.Now().Unix()-entry.created > 3600 {
		delete(t.tokens, nonce)
		return "", false
	}
	return entry.token, true
}

type captchaStore struct {
	mu    sync.Mutex
	items map[string]captchaEntry
}

type captchaEntry struct {
	code    string
	created int64
}

func newCaptchaStore() *captchaStore {
	return &captchaStore{items: map[string]captchaEntry{}}
}

func (s *Server) authGetToken(c *call) (any, error) {
	nonce := c.p.String("nonce", "")
	if nonce == "" {
		nonce = params.MD5(strconv.FormatInt(time.Now().UnixNano(), 10))
	}
	return map[string]any{"token": c.srv.tokens.issue(nonce)}, nil
}

func (s *Server) authGetSession(c *call) (any, error) {
	login := strings.TrimSpace(c.p.Get("login"))
	digest := c.p.Get("digest")
	nonce := c.p.Get("nonce")

	user, err := s.db.UserByDomain(login)
	if err != nil {
		return nil, errAuth
	}
	if user.Password != "" && c.p.Get("password") != "" && c.p.Get("password") != user.Password {
		return nil, errAuth
	}

	if s.cfg.RequireSignature {
		token, ok := s.tokens.get(nonce)
		if !ok {
			return nil, errf(4, "Incorrect signature.")
		}
		sig := params.MD5(login + ":vk.com:" + user.Password)
		expected := params.MD5(s.cfg.AppSecret + ":" + nonce + ":" + token + ":" + sig)
		if digest != expected {
			return nil, errf(4, "Incorrect signature.")
		}
	}

	session, err := s.createSession(user.UID)
	if err != nil {
		return nil, err
	}
	s.db.SetUserOnline(user.UID, true)
	return map[string]any{
		"auth":       "success",
		"id":         user.UID,
		"sid":        session.SID,
		"secret":     session.Secret,
		"first_name": user.FirstName,
		"last_name":  user.LastName,
		"photo":      c.userPhoto(user.UID, "rec"),
	}, nil
}

func (s *Server) createSession(uid int) (model.Session, error) {
	seed := strconv.FormatInt(time.Now().UnixNano(), 10)
	session := model.Session{
		SID:     params.MD5("sid:" + seed),
		UID:     uid,
		Secret:  params.MD5("secret:" + seed + ":vkemu"),
		Created: time.Now().Unix(),
	}
	if err := s.db.SaveSession(session); err != nil {
		return model.Session{}, err
	}
	return session, nil
}

func (s *Server) authLogout(c *call) (any, error) {
	if c.sid != "" {
		s.db.DeleteSession(c.sid)
	}
	if c.uid != 0 {
		s.db.SetUserOnline(c.uid, false)
	}
	return 1, nil
}

func (s *Server) getViewerID(c *call) (any, error) {
	return c.uid, nil
}
