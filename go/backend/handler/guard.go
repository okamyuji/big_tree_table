package handler

import (
	"crypto/subtle"
	"net"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"
)

// The limits mirror rails/app/controllers/api/base_controller.rb so the three
// benchmark targets do the same work per request.
const (
	DefaultRateLimit  = 300
	RateLimitWindow   = time.Minute
	AuthFailureLimit  = 10
	AuthFailureWindow = 15 * time.Minute
	authRealm         = `Basic realm="big_tree_table API"`
	// Past this many tracked IPs, expired failure counters are swept.
	maxTrackedIPs = 10_000
)

type windowCounter struct {
	count     int
	expiresAt time.Time
}

// add bumps the counter by delta in a fixed window that starts at the first
// bump, like ActiveSupport::Cache::MemoryStore#increment with expires_in.
func (c *windowCounter) add(now time.Time, delta int, window time.Duration) int {
	if !now.Before(c.expiresAt) {
		c.count = 0
		c.expiresAt = now.Add(window)
	}
	c.count += delta
	return c.count
}

// Guard applies the API's global rate limit, then HTTP Basic auth with a
// per-IP budget for wrong credentials. Counters live in process memory.
type Guard struct {
	user, pass string
	rateLimit  int
	now        func() time.Time

	mu       sync.Mutex
	global   windowCounter
	failures map[string]*windowCounter
}

func NewGuard(user, pass string, rateLimit int) *Guard {
	return &Guard{user: user, pass: pass, rateLimit: rateLimit, now: time.Now, failures: map[string]*windowCounter{}}
}

// NewGuardFromEnv reads API_BASIC_AUTH_USER / API_BASIC_AUTH_PASSWORD and
// API_RATE_LIMIT. Missing credentials make every request fail with 401.
func NewGuardFromEnv() *Guard {
	return NewGuard(os.Getenv("API_BASIC_AUTH_USER"), os.Getenv("API_BASIC_AUTH_PASSWORD"), rateLimitFromEnv())
}

func rateLimitFromEnv() int {
	n, err := strconv.Atoi(os.Getenv("API_RATE_LIMIT"))
	if err != nil || n <= 0 {
		return DefaultRateLimit
	}
	return n
}

func (g *Guard) Wrap(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if g.bumpGlobal() > g.rateLimit {
			http.Error(w, "Too Many Requests", http.StatusTooManyRequests)
			return
		}
		if g.user == "" || g.pass == "" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		// The browser's first request carries no credentials, so it is neither counted nor blocked.
		if r.Header.Get("Authorization") == "" {
			challenge(w)
			return
		}
		ip := clientIP(r)
		if g.bumpFailures(ip, 1) > AuthFailureLimit {
			http.Error(w, "Too Many Requests", http.StatusTooManyRequests)
			return
		}
		if !g.credentialsMatch(r) {
			challenge(w)
			return
		}
		g.bumpFailures(ip, -1)
		next.ServeHTTP(w, r)
	})
}

func (g *Guard) credentialsMatch(r *http.Request) bool {
	user, pass, ok := r.BasicAuth()
	if !ok {
		return false
	}
	// Both comparisons always run so the timing does not reveal which part was wrong.
	userOK := subtle.ConstantTimeCompare([]byte(user), []byte(g.user))
	passOK := subtle.ConstantTimeCompare([]byte(pass), []byte(g.pass))
	return userOK&passOK == 1
}

func (g *Guard) bumpGlobal() int {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.global.add(g.now(), 1, RateLimitWindow)
}

func (g *Guard) bumpFailures(ip string, delta int) int {
	g.mu.Lock()
	defer g.mu.Unlock()
	now := g.now()
	c, ok := g.failures[ip]
	if !ok {
		if len(g.failures) >= maxTrackedIPs {
			g.sweepExpired(now)
		}
		c = &windowCounter{}
		g.failures[ip] = c
	}
	return c.add(now, delta, AuthFailureWindow)
}

func (g *Guard) sweepExpired(now time.Time) {
	for ip, c := range g.failures {
		if !now.Before(c.expiresAt) {
			delete(g.failures, ip)
		}
	}
}

func (g *Guard) trackedIPs() int {
	g.mu.Lock()
	defer g.mu.Unlock()
	return len(g.failures)
}

func challenge(w http.ResponseWriter) {
	w.Header().Set("WWW-Authenticate", authRealm)
	http.Error(w, "HTTP Basic: Access denied.", http.StatusUnauthorized)
}

func clientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
