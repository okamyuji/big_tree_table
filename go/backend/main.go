package main

import (
	"log"
	"net/http"
	"time"

	"bigtable-backend/db"
	"bigtable-backend/handler"
	"bigtable-backend/middleware"
	"bigtable-backend/repository"
	"bigtable-backend/service"
)

func main() {
	conn, err := db.Connect()
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer conn.Close()

	repo := repository.NewMySQLOrderRepository(conn)
	svc := service.NewOrderService(repo)

	mux := http.NewServeMux()
	mux.HandleFunc("/api/orders", handler.OrdersHandler(svc))
	mux.HandleFunc("/api/order-tree", handler.OrderTreeHandler(svc))

	wrapped := middleware.CORS(mux)

	// Slowloris DoS 対策のためタイムアウトを明示的に設定する。
	// http.ListenAndServe をそのまま使うと各タイムアウトが 0 (= 無制限) となり、
	// ヘッダ／ボディを極端に遅く送る攻撃でハンドラ枠を簡単に枯渇させられる。
	srv := &http.Server{
		Addr:              ":8080",
		Handler:           wrapped,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	log.Println("Server starting on :8080")
	if err := srv.ListenAndServe(); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
