package model

import (
	"fmt"
	"strings"
	"time"
)

type Order struct {
	ID           int64     `json:"id"`
	OrderNumber  string    `json:"order_number"`
	OrderType    string    `json:"order_type"`
	OrderDate    string    `json:"order_date"`
	CustomerName string    `json:"customer_name"`
	CustomerCode string    `json:"customer_code"`
	ProductName  string    `json:"product_name"`
	ProductCode  string    `json:"product_code"`
	Quantity     int       `json:"quantity"`
	UnitPrice    float64   `json:"unit_price"`
	TotalAmount  float64   `json:"total_amount"`
	Status       string    `json:"status"`
	DeliveryDate string    `json:"delivery_date"`
	Notes        string    `json:"notes"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type QueryParams struct {
	Page         int
	PerPage      int
	Sort         string
	Order        string
	OrderType    string
	Status       string
	CustomerName string
	ProductName  string
	DateFrom     string
	DateTo       string
}

type OrdersResponse struct {
	Data       []Order `json:"data"`
	Total      int64   `json:"total"`
	Page       int     `json:"page"`
	PerPage    int     `json:"per_page"`
	TotalPages int     `json:"total_pages"`
}

var allowedSortColumns = map[string]bool{
	"id": true, "order_number": true, "order_type": true,
	"order_date": true, "customer_name": true, "customer_code": true,
	"product_name": true, "product_code": true, "quantity": true,
	"unit_price": true, "total_amount": true, "status": true,
	"delivery_date": true, "created_at": true,
}

const offsetThreshold = 10000

func BuildQuery(p QueryParams) (string, []any) {
	where, args := buildWhere(p)
	sortCol := "id"
	if allowedSortColumns[p.Sort] {
		sortCol = p.Sort
	}
	sortOrder := "ASC"
	if strings.ToLower(p.Order) == "desc" {
		sortOrder = "DESC"
	}
	offset := (p.Page - 1) * p.PerPage

	// OFFSETが大きい場合はdeferred joinで最適化する。
	// 内側のサブクエリがインデックスのみをスキャンしてIDを特定し、
	// 外側のJOINで該当行のフルデータを取得する。
	if offset >= offsetThreshold {
		query := fmt.Sprintf(
			"SELECT o.id, o.order_number, o.order_type, o.order_date, o.customer_name, o.customer_code, o.product_name, o.product_code, o.quantity, o.unit_price, o.total_amount, o.status, o.delivery_date, o.notes, o.created_at, o.updated_at FROM orders o INNER JOIN (SELECT id FROM orders %s ORDER BY %s %s LIMIT %d OFFSET %d) sub ON o.id = sub.id ORDER BY o.%s %s",
			where, sortCol, sortOrder, p.PerPage, offset, sortCol, sortOrder,
		)
		return query, args
	}

	query := fmt.Sprintf(
		"SELECT id, order_number, order_type, order_date, customer_name, customer_code, product_name, product_code, quantity, unit_price, total_amount, status, delivery_date, notes, created_at, updated_at FROM orders %s ORDER BY %s %s LIMIT %d OFFSET %d",
		where, sortCol, sortOrder, p.PerPage, offset,
	)
	return query, args
}

func BuildCountQuery(p QueryParams) (string, []any) {
	where, args := buildWhere(p)
	return fmt.Sprintf("SELECT COUNT(*) FROM orders %s", where), args
}

func buildWhere(p QueryParams) (string, []any) {
	var conditions []string
	var args []any
	if p.OrderType != "" {
		conditions = append(conditions, "order_type = ?")
		args = append(args, p.OrderType)
	}
	if p.Status != "" {
		conditions = append(conditions, "status = ?")
		args = append(args, p.Status)
	}
	if p.CustomerName != "" {
		conditions = append(conditions, "customer_name LIKE ?")
		args = append(args, "%"+p.CustomerName+"%")
	}
	if p.ProductName != "" {
		conditions = append(conditions, "product_name LIKE ?")
		args = append(args, "%"+p.ProductName+"%")
	}
	if p.DateFrom != "" {
		conditions = append(conditions, "order_date >= ?")
		args = append(args, p.DateFrom)
	}
	if p.DateTo != "" {
		conditions = append(conditions, "order_date <= ?")
		args = append(args, p.DateTo)
	}
	if len(conditions) == 0 {
		return "", args
	}
	return "WHERE " + strings.Join(conditions, " AND "), args
}
