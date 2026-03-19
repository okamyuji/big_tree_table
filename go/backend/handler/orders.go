package handler

import (
	"encoding/json"
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

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
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
	if p.Page < 1 {
		p.Page = 1
	}
	if p.PerPage < 1 {
		p.PerPage = 1
	}
	if p.PerPage > 100 {
		p.PerPage = 100
	}
	if p.Sort == "" {
		p.Sort = "id"
	}
	if p.Order == "" {
		p.Order = "asc"
	}
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
