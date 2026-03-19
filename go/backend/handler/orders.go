package handler

import (
	"database/sql"
	"encoding/json"
	"math"
	"net/http"
	"strconv"

	"bigtable-backend/model"
)

func OrdersHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		params := parseQueryParams(r)

		// Count query
		countQuery, countArgs := model.BuildCountQuery(params)
		var total int64
		if err := db.QueryRow(countQuery, countArgs...).Scan(&total); err != nil {
			http.Error(w, "Database error", http.StatusInternalServerError)
			return
		}

		// Data query
		dataQuery, dataArgs := model.BuildQuery(params)
		rows, err := db.Query(dataQuery, dataArgs...)
		if err != nil {
			http.Error(w, "Database error", http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		orders := make([]model.Order, 0)
		for rows.Next() {
			var o model.Order
			if err := rows.Scan(
				&o.ID, &o.OrderNumber, &o.OrderType, &o.OrderDate,
				&o.CustomerName, &o.CustomerCode, &o.ProductName, &o.ProductCode,
				&o.Quantity, &o.UnitPrice, &o.TotalAmount, &o.Status,
				&o.DeliveryDate, &o.Notes, &o.CreatedAt, &o.UpdatedAt,
			); err != nil {
				http.Error(w, "Scan error", http.StatusInternalServerError)
				return
			}
			orders = append(orders, o)
		}

		totalPages := int(math.Ceil(float64(total) / float64(params.PerPage)))

		resp := model.OrdersResponse{
			Data:       orders,
			Total:      total,
			Page:       params.Page,
			PerPage:    params.PerPage,
			TotalPages: totalPages,
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
