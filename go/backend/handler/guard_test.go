package handler

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"
)

type clock struct{ t time.Time }

func (c *clock) now() time.Time { return c.t }

func newTestGuard(user, pass string, rateLimit int) (*Guard, *clock) {
	c := &clock{t: time.Date(2026, 11, 26, 0, 0, 0, 0, time.UTC)}
	g := NewGuard(user, pass, rateLimit)
	g.now = c.now
	return g, c
}

func call(g *Guard, ip, user, pass string) *httptest.ResponseRecorder {
	ok := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })
	r := httptest.NewRequest("GET", "/api/v1/orders", nil)
	r.RemoteAddr = ip + ":40000"
	if user != "" || pass != "" {
		r.SetBasicAuth(user, pass)
	}
	rec := httptest.NewRecorder()
	g.Wrap(ok).ServeHTTP(rec, r)
	return rec
}

const realmHeader = `Basic realm="big_tree_table API"`

func TestGuard_FailsClosedWhenCredentialsUnset(t *testing.T) {
	g, _ := newTestGuard("", "", 300)
	rec := call(g, "203.0.113.1", "a", "b")
	if rec.Code != http.StatusUnauthorized || rec.Header().Get("WWW-Authenticate") != "" {
		t.Errorf("status=%d WWW-Authenticate=%q", rec.Code, rec.Header().Get("WWW-Authenticate"))
	}
}

func TestGuard_ChallengesWithoutCountingWhenNoCredentials(t *testing.T) {
	g, _ := newTestGuard("viewer", "right", 300)
	for range AuthFailureLimit + 5 {
		rec := call(g, "203.0.113.2", "", "")
		if rec.Code != http.StatusUnauthorized || rec.Header().Get("WWW-Authenticate") != realmHeader {
			t.Fatalf("status=%d WWW-Authenticate=%q", rec.Code, rec.Header().Get("WWW-Authenticate"))
		}
	}
	if rec := call(g, "203.0.113.2", "viewer", "right"); rec.Code != http.StatusOK {
		t.Errorf("right credentials after bare requests: status=%d", rec.Code)
	}
}

func TestGuard_AcceptsRightCredentials(t *testing.T) {
	g, _ := newTestGuard("viewer", "right", 300)
	if rec := call(g, "203.0.113.3", "viewer", "right"); rec.Code != http.StatusOK {
		t.Errorf("status=%d", rec.Code)
	}
	for _, c := range [][2]string{{"viewer", "wrong"}, {"other", "right"}, {"viewe", "right"}, {"viewer", "rightx"}} {
		if rec := call(g, "203.0.113.3", c[0], c[1]); rec.Code != http.StatusUnauthorized || rec.Header().Get("WWW-Authenticate") != realmHeader {
			t.Errorf("%v: status=%d", c, rec.Code)
		}
	}
}

func TestGuard_BlocksAnIPAfterTooManyFailuresEvenWithRightPassword(t *testing.T) {
	g, c := newTestGuard("viewer", "right", 300)
	for i := range AuthFailureLimit {
		if rec := call(g, "203.0.113.4", "viewer", "wrong"); rec.Code != http.StatusUnauthorized {
			t.Fatalf("attempt %d: status=%d", i+1, rec.Code)
		}
	}
	if rec := call(g, "203.0.113.4", "viewer", "right"); rec.Code != http.StatusTooManyRequests {
		t.Errorf("after %d failures: status=%d, want 429", AuthFailureLimit, rec.Code)
	}
	if rec := call(g, "203.0.113.5", "viewer", "right"); rec.Code != http.StatusOK {
		t.Errorf("another IP: status=%d", rec.Code)
	}
	c.t = c.t.Add(AuthFailureWindow)
	if rec := call(g, "203.0.113.4", "viewer", "right"); rec.Code != http.StatusOK {
		t.Errorf("after the window: status=%d", rec.Code)
	}
}

func TestGuard_SuccessGivesBackItsCount(t *testing.T) {
	g, _ := newTestGuard("viewer", "right", 300)
	for range AuthFailureLimit - 1 {
		call(g, "203.0.113.6", "viewer", "wrong")
	}
	for i := range 20 {
		if rec := call(g, "203.0.113.6", "viewer", "right"); rec.Code != http.StatusOK {
			t.Fatalf("success %d: status=%d", i+1, rec.Code)
		}
	}
	if rec := call(g, "203.0.113.6", "viewer", "wrong"); rec.Code != http.StatusUnauthorized {
		t.Errorf("10th failure: status=%d, want 401", rec.Code)
	}
	if rec := call(g, "203.0.113.6", "viewer", "right"); rec.Code != http.StatusTooManyRequests {
		t.Errorf("11th attempt: status=%d, want 429", rec.Code)
	}
}

func TestGuard_GlobalRateLimitCountsEveryRequestInAFixedWindow(t *testing.T) {
	g, c := newTestGuard("viewer", "right", 3)
	codes := []int{
		call(g, "203.0.113.7", "", "").Code,
		call(g, "203.0.113.8", "viewer", "wrong").Code,
		call(g, "203.0.113.9", "viewer", "right").Code,
		call(g, "203.0.113.9", "viewer", "right").Code,
	}
	want := []int{http.StatusUnauthorized, http.StatusUnauthorized, http.StatusOK, http.StatusTooManyRequests}
	for i := range want {
		if codes[i] != want[i] {
			t.Errorf("request %d: status=%d, want %d", i+1, codes[i], want[i])
		}
	}
	c.t = c.t.Add(RateLimitWindow - time.Nanosecond)
	if rec := call(g, "203.0.113.9", "viewer", "right"); rec.Code != http.StatusTooManyRequests {
		t.Errorf("same window: status=%d", rec.Code)
	}
	c.t = c.t.Add(time.Nanosecond)
	if rec := call(g, "203.0.113.9", "viewer", "right"); rec.Code != http.StatusOK {
		t.Errorf("next window: status=%d", rec.Code)
	}
}

func TestGuard_DropsExpiredFailureCountersWhenTheTableGrows(t *testing.T) {
	g, c := newTestGuard("viewer", "right", 1<<30)
	for i := range maxTrackedIPs {
		call(g, "198.51.100."+strconv.Itoa(i), "viewer", "wrong")
	}
	c.t = c.t.Add(AuthFailureWindow)
	call(g, "203.0.113.10", "viewer", "wrong")
	if n := g.trackedIPs(); n != 1 {
		t.Errorf("tracked IPs = %d, want 1", n)
	}
}

func TestRateLimitFromEnv(t *testing.T) {
	cases := map[string]int{"": 300, "1000000": 1000000, "abc": 300, "0": 300, "-1": 300}
	for v, want := range cases {
		t.Setenv("API_RATE_LIMIT", v)
		if got := rateLimitFromEnv(); got != want {
			t.Errorf("API_RATE_LIMIT=%q: %d, want %d", v, got, want)
		}
	}
}
