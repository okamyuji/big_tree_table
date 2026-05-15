package model

import (
	"fmt"
	"regexp"
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

// MaxPage は (Page-1)*PerPage の整数オーバーフローおよび過大OFFSETによる
// DoS を防ぐための上限。100 万件 × PerPage 100 = OFFSET 1 億までは到達可能。
const MaxPage = 1_000_000

// MaxPerPage は 1 ページあたりの最大件数。
const MaxPerPage = 100

// Normalize clamps QueryParams to safe ranges so that downstream BuildQuery /
// BuildCountQuery にユーザー入力由来の不正値 (負数 OFFSET の原因となる過大 Page、
// 範囲外 PerPage、空の Sort/Order) が流入しないことを保証する。
// QueryParams を組み立てた呼び出し側は BuildQuery 前に必ずこのメソッドを呼ぶこと。
func (p *QueryParams) Normalize() {
	if p.Page < 1 {
		p.Page = 1
	}
	if p.Page > MaxPage {
		p.Page = MaxPage
	}
	if p.PerPage < 1 {
		p.PerPage = 1
	}
	if p.PerPage > MaxPerPage {
		p.PerPage = MaxPerPage
	}
	if p.Sort == "" {
		p.Sort = "id"
	}
	if p.Order == "" {
		p.Order = "asc"
	}
}

type OrdersResponse struct {
	Data       []Order `json:"data"`
	Total      int64   `json:"total"`
	Page       int     `json:"page"`
	PerPage    int     `json:"per_page"`
	TotalPages int     `json:"total_pages"`
}

type TreeSummary struct {
	OrderCount  int      `json:"order_count"`
	Quantity    int      `json:"quantity"`
	TotalAmount float64  `json:"total_amount"`
	Statuses    []string `json:"statuses"`
}

type OrderTreeNode struct {
	ID       string          `json:"id"`
	Kind     string          `json:"kind"`
	Depth    int             `json:"depth"`
	Label    string          `json:"label"`
	Order    *Order          `json:"order,omitempty"`
	Summary  TreeSummary     `json:"summary"`
	Children []OrderTreeNode `json:"children"`
}

type OrderTreeResponse struct {
	Data       []OrderTreeNode `json:"data"`
	Total      int64           `json:"total"`
	Page       int             `json:"page"`
	PerPage    int             `json:"per_page"`
	TotalPages int             `json:"total_pages"`
}

// sanitizeSortColumn maps a user-supplied sort key to a literal column name.
// 各caseでリテラル文字列を返すことで、ユーザー入力からSQL文へのテイント
// 伝播を遮断し、CodeQLのSQLインジェクション検出に対しても安全とする。
func sanitizeSortColumn(s string) string {
	switch s {
	case "id":
		return "id"
	case "order_number":
		return "order_number"
	case "order_type":
		return "order_type"
	case "order_date":
		return "order_date"
	case "customer_name":
		return "customer_name"
	case "customer_code":
		return "customer_code"
	case "product_name":
		return "product_name"
	case "product_code":
		return "product_code"
	case "quantity":
		return "quantity"
	case "unit_price":
		return "unit_price"
	case "total_amount":
		return "total_amount"
	case "status":
		return "status"
	case "delivery_date":
		return "delivery_date"
	case "created_at":
		return "created_at"
	}
	return "id"
}

// sanitizeSortOrder returns a literal "ASC"/"DESC" string to defeat taint tracking.
func sanitizeSortOrder(s string) string {
	if strings.ToLower(s) == "desc" {
		return "DESC"
	}
	return "ASC"
}

const offsetThreshold = 10000

func BuildQuery(p QueryParams) (string, []any) {
	where, args := buildWhere(p)
	sortCol := sanitizeSortColumn(p.Sort)
	sortOrder := sanitizeSortOrder(p.Order)
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

func BuildOrderTree(orders []Order) []OrderTreeNode {
	customers := make([]OrderTreeNode, 0)
	customerIndexes := make(map[string]int)
	productIndexes := make(map[string]map[string]int)

	for _, order := range orders {
		customerID := "customer:" + order.CustomerCode
		customerIndex, ok := customerIndexes[customerID]
		if !ok {
			customers = append(customers, OrderTreeNode{
				ID:       customerID,
				Kind:     "customer",
				Depth:    0,
				Label:    order.CustomerName,
				Summary:  TreeSummary{Statuses: []string{}},
				Children: []OrderTreeNode{},
			})
			customerIndex = len(customers) - 1
			customerIndexes[customerID] = customerIndex
			productIndexes[customerID] = make(map[string]int)
		}

		appendSummary(&customers[customerIndex].Summary, order)

		productID := customerID + ":product:" + order.ProductCode
		productIndex, ok := productIndexes[customerID][productID]
		if !ok {
			customers[customerIndex].Children = append(customers[customerIndex].Children, OrderTreeNode{
				ID:       productID,
				Kind:     "product",
				Depth:    1,
				Label:    order.ProductName,
				Summary:  TreeSummary{Statuses: []string{}},
				Children: []OrderTreeNode{},
			})
			productIndex = len(customers[customerIndex].Children) - 1
			productIndexes[customerID][productID] = productIndex
		}

		productNode := &customers[customerIndex].Children[productIndex]
		appendSummary(&productNode.Summary, order)
		orderCopy := order
		productNode.Children = append(productNode.Children, OrderTreeNode{
			ID:       fmt.Sprintf("order:%d", order.ID),
			Kind:     "order",
			Depth:    2,
			Label:    order.OrderNumber,
			Order:    &orderCopy,
			Summary:  TreeSummary{OrderCount: 1, Quantity: order.Quantity, TotalAmount: order.TotalAmount, Statuses: []string{order.Status}},
			Children: []OrderTreeNode{},
		})
	}

	return customers
}

func appendSummary(summary *TreeSummary, order Order) {
	summary.OrderCount++
	summary.Quantity += order.Quantity
	summary.TotalAmount += order.TotalAmount
	for _, status := range summary.Statuses {
		if status == order.Status {
			return
		}
	}
	summary.Statuses = append(summary.Statuses, order.Status)
}

// datePattern validates YYYY-MM-DD format
var datePattern = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)

// escapeLike escapes LIKE wildcards (% and _) in user input
func escapeLike(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `%`, `\%`)
	s = strings.ReplaceAll(s, `_`, `\_`)
	return s
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
		args = append(args, "%"+escapeLike(p.CustomerName)+"%")
	}
	if p.ProductName != "" {
		conditions = append(conditions, "product_name LIKE ?")
		args = append(args, "%"+escapeLike(p.ProductName)+"%")
	}
	if p.DateFrom != "" && datePattern.MatchString(p.DateFrom) {
		conditions = append(conditions, "order_date >= ?")
		args = append(args, p.DateFrom)
	}
	if p.DateTo != "" && datePattern.MatchString(p.DateTo) {
		conditions = append(conditions, "order_date <= ?")
		args = append(args, p.DateTo)
	}
	if len(conditions) == 0 {
		return "", args
	}
	return "WHERE " + strings.Join(conditions, " AND "), args
}
