package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
)

// pingTimeout bounds the startup check: an unreachable host otherwise blocks for the OS TCP timeout (75s on macOS).
const pingTimeout = 5 * time.Second

// Postgres reports whether DB_DRIVER=postgres selects PostgreSQL (pgx, PG_URL) instead of MySQL.
func Postgres() bool { return os.Getenv("DB_DRIVER") == "postgres" }

func Connect() (*sql.DB, error) {
	db, err := open()
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(envInt("DB_MAX_CONNS", 25))
	db.SetMaxIdleConns(envInt("DB_MAX_IDLE_CONNS", 5))
	ctx, cancel := context.WithTimeout(context.Background(), pingTimeout)
	defer cancel()
	return db, db.PingContext(ctx)
}

func open() (*sql.DB, error) {
	if !Postgres() {
		return sql.Open("mysql", mysqlDSN())
	}
	url := os.Getenv("PG_URL")
	if url == "" {
		return nil, errors.New("DB_DRIVER=postgres needs PG_URL")
	}
	cfg, err := postgresConfig(url)
	if err != nil {
		return nil, err
	}
	return stdlib.OpenDB(*cfg), nil
}

// postgresConfig plans every query with its actual values. pgx's default
// caches prepared statements, and after a few runs PostgreSQL may switch them
// to a generic plan: for the LIKE + date-range filter that plan measured 77 ms
// against 0.9 ms. Rails and snails inline the values, so they never hit it.
// A default_query_exec_mode in PG_URL still wins.
func postgresConfig(url string) (*pgx.ConnConfig, error) {
	cfg, err := pgx.ParseConfig(url)
	if err != nil {
		return nil, err
	}
	if !strings.Contains(url, "default_query_exec_mode") {
		cfg.DefaultQueryExecMode = pgx.QueryExecModeExec
	}
	return cfg, nil
}

func mysqlDSN() string {
	host := getEnv("DB_HOST", "localhost")
	port := getEnv("DB_PORT", "3306")
	user := getEnv("DB_USER", "root")
	// パスワードはハードコードしない。未設定時は空文字となり、
	// 環境に応じて env / .env で必ず設定する運用とする。
	pass := os.Getenv("DB_PASS")
	name := getEnv("DB_NAME", "bigtable")
	return fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true&charset=utf8mb4", user, pass, host, port, name)
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// envInt reads a positive integer, falling back when the value is unset or invalid.
func envInt(key string, fallback int) int {
	n, err := strconv.Atoi(os.Getenv(key))
	if err != nil || n <= 0 {
		return fallback
	}
	return n
}
