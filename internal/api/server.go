package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"vkemu/internal/config"
	"vkemu/internal/longpoll"
	"vkemu/internal/params"
	"vkemu/internal/ratelimit"
	"vkemu/internal/store"
)

type Error struct {
	Code    int
	Msg     string
	Captcha bool
}

func (e *Error) Error() string {
	return fmt.Sprintf("api error %d: %s", e.Code, e.Msg)
}

func errf(code int, format string, args ...any) *Error {
	return &Error{Code: code, Msg: fmt.Sprintf(format, args...)}
}

var (
	errAuth      = errf(5, "User authorization failed: invalid session.")
	errSignature = errf(4, "Incorrect signature.")
	errMethod    = errf(3, "Unknown method passed.")
	errParam     = errf(100, "One of the parameters specified was missing or invalid.")
	errAccess    = errf(15, "Access denied.")
	errNotFound  = errf(104, "Not found.")
)

type handler func(*call) (any, error)

type Server struct {
	cfg     *config.Config
	db      *store.Store
	hub     *longpoll.Hub
	tokens  *tokenStore
	captcha *captchaStore
	routes  map[string]handler
	mux     *http.ServeMux
	limit   *ratelimit.Limiter
}

func NewServer(cfg *config.Config, db *store.Store) *Server {
	s := &Server{
		cfg:     cfg,
		db:      db,
		hub:     longpoll.New(),
		tokens:  newTokenStore(),
		captcha: newCaptchaStore(),
		mux:     http.NewServeMux(),
		limit: ratelimit.New(ratelimit.Options{
			Rate:       cfg.RateLimit,
			Burst:      cfg.RateBurst,
			MaxConns:   cfg.MaxConns,
			TrustProxy: cfg.TrustProxy,
			Skip:       func(r *http.Request) bool { return r.URL.Path == "/longpoll" },
		}),
	}
	s.routes = s.buildRoutes()
	s.mux.HandleFunc("/api.php", s.handleAPI)
	s.mux.HandleFunc("/longpoll", s.handleLongPoll)
	s.mux.HandleFunc("/media/", s.handleMedia)
	s.mux.HandleFunc("/upload/", s.handleUpload)
	s.mux.HandleFunc("/captcha", s.handleCaptcha)
	s.mux.HandleFunc("/captcha.png", s.handleCaptcha)
	s.mux.HandleFunc("/", s.handleIndex)
	return s
}

func (s *Server) Handler() http.Handler {
	return s.limit.Middleware(s.mux)
}

func (s *Server) Hub() *longpoll.Hub {
	return s.hub
}

type call struct {
	srv    *Server
	p      params.Params
	uid    int
	sid    string
	secret string
	scheme string
	host   string
}

func (c *call) viewer() int {
	return c.uid
}

func (c *call) abs(path string) string {
	if path == "" {
		return ""
	}
	if strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://") {
		return path
	}
	if c.host != "" {
		scheme := c.scheme
		if scheme == "" {
			scheme = "http"
		}
		return scheme + "://" + c.host + path
	}
	return c.srv.abs(path)
}

func (s *Server) abs(path string) string {
	if path == "" {
		return ""
	}
	if strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://") {
		return path
	}
	return s.cfg.BaseURL + path
}

func (s *Server) handleAPI(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		writeError(w, errParam)
		return
	}
	p := params.FromValues(r.Form)
	if q := r.URL.Query(); len(q) > 0 {
		for k, v := range q {
			if _, ok := p[k]; !ok && len(v) > 0 {
				p[k] = v[0]
			}
		}
	}

	c := &call{srv: s, p: p}
	c.scheme = requestScheme(r)
	c.host = r.Host
	if sid := p.Get("sid"); sid != "" {
		c.sid = sid
		if session, err := s.db.Session(sid); err == nil {
			c.uid = session.UID
			c.secret = session.Secret
		}
	}

	method := strings.TrimSpace(p.Get("method"))
	if method == "" {
		writeError(w, errParam)
		return
	}

	if s.cfg.RequireSignature && !publicMethod(method) {
		if c.uid == 0 {
			writeError(w, errAuth)
			return
		}
		if p.Get("sig") != p.Signature(c.uid, c.secret) {
			writeError(w, errSignature)
			return
		}
	}

	value, err := s.invoke(c, method, p)
	if err != nil {
		writeError(w, err)
		return
	}
	writeResponse(w, value)
}

func (s *Server) invoke(c *call, method string, p params.Params) (any, error) {
	h, ok := s.routes[method]
	if !ok {
		return nil, errMethod
	}
	nested := &call{srv: s, p: p, uid: c.uid, sid: c.sid, secret: c.secret, scheme: c.scheme, host: c.host}
	return h(nested)
}

func requestScheme(r *http.Request) string {
	if proto := r.Header.Get("X-Forwarded-Proto"); proto != "" {
		return proto
	}
	if r.TLS != nil {
		return "https"
	}
	return "http"
}

func (s *Server) longPollHost(c *call) string {
	if c.host != "" {
		return c.host + "/longpoll"
	}
	return s.cfg.LongPollHost
}

func publicMethod(method string) bool {
	switch method {
	case "auth.getTokenSecure", "auth.getSessionSecure", "auth.getToken", "auth.getSession",
		"auth.login", "auth.signup", "captcha.force":
		return true
	}
	return false
}

func writeResponse(w http.ResponseWriter, value any) {
	writeJSON(w, map[string]any{"response": value})
}

func writeError(w http.ResponseWriter, err error) {
	apiErr, ok := err.(*Error)
	if !ok {
		apiErr = errf(1, "%s", err.Error())
	}
	body := map[string]any{
		"error_code": apiErr.Code,
		"error_msg":  apiErr.Msg,
	}
	if apiErr.Captcha {
		body["captcha_sid"] = "1"
		body["captcha_img"] = "/captcha.png"
	}
	writeJSON(w, map[string]any{"error": body})
}

func writeJSON(w http.ResponseWriter, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(value); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	fmt.Fprintf(w, "vk api")
}
