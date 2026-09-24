package ratelimit

import (
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

type Options struct {
	Rate       float64
	Burst      float64
	MaxConns   int
	TrustProxy bool
	Skip       func(*http.Request) bool
	Now        func() time.Time
}

type client struct {
	tokens float64
	last   time.Time
}

type Limiter struct {
	opts     Options
	mu       sync.Mutex
	clients  map[string]*client
	inflight chan struct{}
	stop     chan struct{}
	once     sync.Once
}

func New(opts Options) *Limiter {
	if opts.Now == nil {
		opts.Now = time.Now
	}
	if opts.Burst < 1 {
		opts.Burst = 1
	}
	l := &Limiter{
		opts:    opts,
		clients: make(map[string]*client),
		stop:    make(chan struct{}),
	}
	if opts.MaxConns > 0 {
		l.inflight = make(chan struct{}, opts.MaxConns)
	}
	go l.cleanup()
	return l
}

func (l *Limiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if l.opts.Skip != nil && l.opts.Skip(r) {
			next.ServeHTTP(w, r)
			return
		}
		if l.opts.Rate > 0 && !l.allow(l.key(r)) {
			writeTooMany(w)
			return
		}
		if l.inflight != nil {
			select {
			case l.inflight <- struct{}{}:
				defer func() { <-l.inflight }()
			default:
				w.Header().Set("Retry-After", "1")
				http.Error(w, "server busy", http.StatusServiceUnavailable)
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}

func (l *Limiter) Close() {
	l.once.Do(func() { close(l.stop) })
}

func (l *Limiter) allow(key string) bool {
	now := l.opts.Now()
	l.mu.Lock()
	defer l.mu.Unlock()

	c, ok := l.clients[key]
	if !ok {
		l.clients[key] = &client{tokens: l.opts.Burst - 1, last: now}
		return true
	}
	if elapsed := now.Sub(c.last).Seconds(); elapsed > 0 {
		c.tokens += elapsed * l.opts.Rate
		if c.tokens > l.opts.Burst {
			c.tokens = l.opts.Burst
		}
		c.last = now
	}
	if c.tokens < 1 {
		return false
	}
	c.tokens--
	return true
}

func (l *Limiter) key(r *http.Request) string {
	if l.opts.TrustProxy {
		if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
			if first := strings.TrimSpace(strings.Split(forwarded, ",")[0]); first != "" {
				return first
			}
		}
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

func (l *Limiter) cleanup() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-l.stop:
			return
		case <-ticker.C:
			cutoff := l.opts.Now().Add(-3 * time.Minute)
			l.mu.Lock()
			for key, c := range l.clients {
				if c.last.Before(cutoff) {
					delete(l.clients, key)
				}
			}
			l.mu.Unlock()
		}
	}
}

func writeTooMany(w http.ResponseWriter) {
	w.Header().Set("Retry-After", "1")
	http.Error(w, "too many requests", http.StatusTooManyRequests)
}
