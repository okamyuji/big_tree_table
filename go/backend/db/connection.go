package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"strconv"
	"time"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/jackc/pgx/v5/stdlib"
)

// pingTimeout bounds the startup check: an unreachable host otherwise blocks for the OS TCP timeout (75s on macOS).
const pingTimeout = 5 * time.Second

// Postgres reports whether DB_DRIVER=postgres selects PostgreSQL (pgx, PG_URL) instead of MySQL.
func Postgres() bool { return os.Getenv("DB_DRIVER") == "postgres" }

func Connect() (*sql.DB, error) {
	driver, dsn := "mysql", mysqlDSN()
	if Postgres() {
		driver, dsn = "pgx", os.Getenv("PG_URL")
		if dsn == "" {
			return nil, errors.New("DB_DRIVER=postgres needs PG_URL")
		}
	}
	db, err := sql.Open(driver, dsn)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(envInt("DB_MAX_CONNS", 25))
	db.SetMaxIdleConns(envInt("DB_MAX_IDLE_CONNS", 5))
	ctx, cancel := context.WithTimeout(context.Background(), pingTimeout)
	defer cancel()
	return db, db.PingContext(ctx)
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
