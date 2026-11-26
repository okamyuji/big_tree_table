package main

import (
	"log"
	"net/http"
	"os"
	"time"

	"bigtable-backend/db"
	"bigtable-backend/handler"
	"bigtable-backend/repository"
	"bigtable-backend/service"
)

func main() {
	conn, err := db.Connect()
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer conn.Close()

	var repo repository.OrderRepository = repository.NewMySQLOrderRepository(conn)
	if db.Postgres() {
		repo = repository.NewPostgresOrderRepository(conn)
	}
	svc := service.NewOrderService(repo)
	guard := handler.NewGuardFromEnv()

	mux := http.NewServeMux()
	mux.HandleFunc("/api/orders", handler.OrdersHandler(svc))
	mux.HandleFunc("/api/order-tree", handler.OrderTreeHandler(svc))
	// /api/v1 matches the Rails API, Basic auth and rate limits included.
	mux.Handle("/api/v1/orders", guard.Wrap(handler.V1OrdersHandler(svc)))
	mux.Handle("/api/v1/orders/tree", guard.Wrap(handler.V1OrderTreeHandler(svc)))

	// Slowloris DoS 対策のためタイムアウトを明示的に設定する。
	// http.ListenAndServe をそのまま使うと各タイムアウトが 0 (= 無制限) となり、
	// ヘッダ／ボディを極端に遅く送る攻撃でハンドラ枠を簡単に枯渇させられる。
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	srv := &http.Server{
		Addr:              ":" + port,
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	log.Println("Server starting on :" + port)
	if err := srv.ListenAndServe(); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
