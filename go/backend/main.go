package main

import (
	"log"
	"net/http"

	"bigtable-backend/db"
	"bigtable-backend/handler"
	"bigtable-backend/middleware"
)

func main() {
	conn, err := db.Connect()
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer conn.Close()

	mux := http.NewServeMux()
	mux.HandleFunc("/api/orders", handler.OrdersHandler(conn))

	wrapped := middleware.CORS(mux)

	log.Println("Server starting on :8080")
	if err := http.ListenAndServe(":8080", wrapped); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
