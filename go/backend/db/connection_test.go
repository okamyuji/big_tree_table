package db

import (
	"testing"
	"time"
)

func TestConnect_FailsWithinPingTimeoutWhenHostDoesNotAnswer(t *testing.T) {
	// 192.0.2.0/24 is TEST-NET-1 (RFC 5737): never routed, so the TCP dial hangs instead of being refused.
	t.Setenv("DB_HOST", "192.0.2.1")
	t.Setenv("DB_PORT", "3306")

	start := time.Now()
	conn, err := Connect()
	elapsed := time.Since(start)
	if conn != nil {
		conn.Close()
	}

	if err == nil {
		t.Fatal("Connect() error = nil, want a timeout error")
	}
	if limit := pingTimeout + 2*time.Second; elapsed > limit {
		t.Errorf("Connect() took %v, want at most %v", elapsed, limit)
	}
}
