package handler

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"

	"bigtable-backend/model"
	"bigtable-backend/service"
)

// OrdersHandler creates an HTTP handler for the orders endpoint
func OrdersHandler(svc service.OrderService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		params := parseQueryParams(r)

		resp, err := svc.GetOrders(params)
		if err != nil {
			http.Error(w, "Database error", http.StatusInternalServerError)
			return
		}

		writeJSON(w, resp)
	}
}

// OrderTreeHandler creates an HTTP handler for the tree table endpoint
func OrderTreeHandler(svc service.OrderService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		params := parseQueryParams(r)

		resp, err := svc.GetOrderTree(params)
		if err != nil {
			http.Error(w, "Database error", http.StatusInternalServerError)
			return
		}

		writeJSON(w, resp)
	}
}

// writeJSON encodes resp as JSON. json.NewEncoder は ResponseWriter に直接
// 書き込むため、エンコード失敗時点で既に HTTP 200 が送信済みでありステータス
// 矛盾は防げない。ここではエラーを握り潰さず log に残し、不完全な
// レスポンス送出を運用上検知可能にすることを目的とする。
func writeJSON(w http.ResponseWriter, resp any) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		log.Printf("failed to encode response: %v", err)
	}
}

func parseQueryParams(r *http.Request) model.QueryParams {
	p := model.QueryParams{
		Page:         parseIntParam(r, "page", 1),
		PerPage:      parseIntParam(r, "per_page", 50),
		Sort:         r.URL.Query().Get("sort"),
		Order:        r.URL.Query().Get("order"),
		OrderType:    r.URL.Query().Get("order_type"),
		Status:       r.URL.Query().Get("status"),
		CustomerName: r.URL.Query().Get("customer_name"),
		ProductName:  r.URL.Query().Get("product_name"),
		DateFrom:     r.URL.Query().Get("date_from"),
		DateTo:       r.URL.Query().Get("date_to"),
	}
	// クランプ/デフォルト適用は model 側に集約し、handler を含むすべての
	// QueryParams 利用箇所で同じ安全範囲が保証されるようにする。
	p.Normalize()
	return p
}

func parseIntParam(r *http.Request, key string, defaultVal int) int {
	s := r.URL.Query().Get(key)
	if s == "" {
		return defaultVal
	}
	v, err := strconv.Atoi(s)
	if err != nil {
		return defaultVal
	}
	return v
}
