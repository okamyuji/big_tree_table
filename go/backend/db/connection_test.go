package db

import (
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
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

func TestConnect_PostgresNeedsPGURL(t *testing.T) {
	t.Setenv("DB_DRIVER", "postgres")
	t.Setenv("PG_URL", "")

	conn, err := Connect()
	if conn != nil || err == nil || err.Error() != "DB_DRIVER=postgres needs PG_URL" {
		t.Fatalf("Connect() = %v, %v; want nil and the PG_URL error", conn, err)
	}
}

func TestConnect_PostgresFailsWithinPingTimeoutWhenHostDoesNotAnswer(t *testing.T) {
	t.Setenv("DB_DRIVER", "postgres")
	t.Setenv("PG_URL", "postgres://u@192.0.2.1:5432/x?connect_timeout=30")

	start := time.Now()
	conn, err := Connect()
	if conn != nil {
		conn.Close()
	}
	if err == nil {
		t.Fatal("Connect() error = nil, want a timeout error")
	}
	if limit := pingTimeout + 2*time.Second; time.Since(start) > limit {
		t.Errorf("Connect() took %v, want at most %v", time.Since(start), limit)
	}
}

func TestPostgres(t *testing.T) {
	for v, want := range map[string]bool{"": false, "mysql": false, "postgres": true, "Postgres": false} {
		t.Setenv("DB_DRIVER", v)
		if got := Postgres(); got != want {
			t.Errorf("DB_DRIVER=%q: Postgres() = %v, want %v", v, got, want)
		}
	}
}

func TestEnvInt(t *testing.T) {
	for v, want := range map[string]int{"": 25, "64": 64, "0": 25, "-3": 25, "x": 25} {
		t.Setenv("DB_MAX_CONNS", v)
		if got := envInt("DB_MAX_CONNS", 25); got != want {
			t.Errorf("DB_MAX_CONNS=%q: %d, want %d", v, got, want)
		}
	}
}

func TestPostgresConfig_PlansEachQueryWithItsValues(t *testing.T) {
	cfg, err := postgresConfig("postgres://u@127.0.0.1:5432/x")
	if err != nil {
		t.Fatal(err)
	}
	if cfg.DefaultQueryExecMode != pgx.QueryExecModeExec {
		t.Errorf("DefaultQueryExecMode = %v, want exec", cfg.DefaultQueryExecMode)
	}
	cfg, err = postgresConfig("postgres://u@127.0.0.1:5432/x?default_query_exec_mode=simple_protocol")
	if err != nil {
		t.Fatal(err)
	}
	if cfg.DefaultQueryExecMode != pgx.QueryExecModeSimpleProtocol {
		t.Errorf("an explicit mode in PG_URL must win, got %v", cfg.DefaultQueryExecMode)
	}
	if _, err := postgresConfig("postgres://u@[::1"); err == nil {
		t.Error("a malformed PG_URL must be an error")
	}
}
