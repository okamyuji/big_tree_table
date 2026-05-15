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

// orderColumns はSELECTとScanで使うカラム順序の単一情報源。
// SELECT列の並びと(*Order).ScanTargets()の並びを必ずここから生成し、
// 物理スキーマの並びとは独立に管理する。物理スキーマの並びに依存しないことで
// migrationでカラムが追加されてもScan順がずれない。
var orderColumns = []string{
	"id", "order_number", "order_type", "order_date",
	"customer_name", "customer_code", "product_name", "product_code",
	"quantity", "unit_price", "total_amount", "status",
	"delivery_date", "notes", "created_at", "updated_at",
}

// orderColumnsSQL は orderColumns をカンマ区切りで結合した SELECT 用文字列。
var orderColumnsSQL = strings.Join(orderColumns, ", ")

// orderColumnsSQLAliased は orderColumns に "o." プレフィックスを付けたバージョン。
// deferred join 時の外側 SELECT に使う。
var orderColumnsSQLAliased = func() string {
	prefixed := make([]string, len(orderColumns))
	for i, c := range orderColumns {
		prefixed[i] = "o." + c
	}
	return strings.Join(prefixed, ", ")
}()

// ScanTargets returns the slice of pointers in the same order as orderColumns
// so that rows.Scan can be invoked without re-specifying columns at each call site.
func (o *Order) ScanTargets() []any {
	return []any{
		&o.ID, &o.OrderNumber, &o.OrderType, &o.OrderDate,
		&o.CustomerName, &o.CustomerCode, &o.ProductName, &o.ProductCode,
		&o.Quantity, &o.UnitPrice, &o.TotalAmount, &o.Status,
		&o.DeliveryDate, &o.Notes, &o.CreatedAt, &o.UpdatedAt,
	}
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

// BuildQuery composes the SELECT clause statically and binds every user-derived
// value through `?` placeholders, including LIMIT / OFFSET. sortCol / sortOrder
// are also user input by name, but they go through whitelists that map to
// compile-time literal strings (`sanitizeSortColumn` / `sanitizeSortOrder`)
// before reaching this assembly step.
func BuildQuery(p QueryParams) (string, []any) {
	where, args := buildWhere(p)
	sortCol := sanitizeSortColumn(p.Sort)
	sortOrder := sanitizeSortOrder(p.Order)
	offset := (p.Page - 1) * p.PerPage

	var b strings.Builder
	// OFFSETが大きい場合はdeferred joinで最適化する。
	// 内側のサブクエリがインデックスのみをスキャンしてIDを特定し、
	// 外側のJOINで該当行のフルデータを取得する。
	if offset >= offsetThreshold {
		b.WriteString("SELECT ")
		b.WriteString(orderColumnsSQLAliased)
		b.WriteString(" FROM orders o INNER JOIN (SELECT id FROM orders")
		if where != "" {
			b.WriteByte(' ')
			b.WriteString(where)
		}
		b.WriteString(" ORDER BY ")
		b.WriteString(sortCol)
		b.WriteByte(' ')
		b.WriteString(sortOrder)
		b.WriteString(" LIMIT ? OFFSET ?) sub ON o.id = sub.id ORDER BY o.")
		b.WriteString(sortCol)
		b.WriteByte(' ')
		b.WriteString(sortOrder)
		args = append(args, p.PerPage, offset)
		return b.String(), args
	}

	b.WriteString("SELECT ")
	b.WriteString(orderColumnsSQL)
	b.WriteString(" FROM orders")
	if where != "" {
		b.WriteByte(' ')
		b.WriteString(where)
	}
	b.WriteString(" ORDER BY ")
	b.WriteString(sortCol)
	b.WriteByte(' ')
	b.WriteString(sortOrder)
	b.WriteString(" LIMIT ? OFFSET ?")
	args = append(args, p.PerPage, offset)
	return b.String(), args
}

func BuildCountQuery(p QueryParams) (string, []any) {
	where, args := buildWhere(p)
	if where == "" {
		return "SELECT COUNT(*) FROM orders", args
	}
	return "SELECT COUNT(*) FROM orders " + where, args
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
