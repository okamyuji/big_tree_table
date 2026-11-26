package handler

import (
	"encoding/json"
	"log"
	"math"
	"net/http"
	"net/url"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	"bigtable-backend/model"
	"bigtable-backend/service"
)

// /api/v1 mirrors the Rails API (rails/app/controllers/api/v1/orders_controller.rb
// and rails/app/models/order.rb): the same parameter normalization, envelope and
// value formats, so the three implementations can be compared byte for byte.
const (
	v1DefaultPerPage = 50
	v1MaxPerPage     = 500
)

var v1DatePattern = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)

func parseV1Params(q url.Values) model.QueryParams {
	perPage := rubyToI(q.Get("per_page"))
	if perPage <= 0 {
		perPage = v1DefaultPerPage
	}
	return model.QueryParams{
		Page:         min(max(rubyToI(q.Get("page")), 1), model.MaxPage),
		PerPage:      min(perPage, v1MaxPerPage),
		Sort:         q.Get("sort"),
		Order:        q.Get("order"),
		OrderType:    q.Get("order_type"),
		Status:       q.Get("status"),
		CustomerName: q.Get("customer_name"),
		ProductName:  q.Get("product_name"),
		DateFrom:     isoDateOrEmpty(q.Get("date_from")),
		DateTo:       isoDateOrEmpty(q.Get("date_to")),
	}
}

// rubyToI follows Ruby's String#to_i: leading whitespace, an optional sign,
// then digits with single underscores between them; anything else ends the
// number. The result saturates instead of overflowing.
func rubyToI(s string) int {
	const limit = 1 << 40
	m := rubyIntPrefix.FindStringSubmatch(s)
	if m == nil {
		return 0
	}
	n, err := strconv.Atoi(strings.ReplaceAll(m[2], "_", ""))
	if err != nil || n > limit { // the digits always parse, so err means overflow
		n = limit
	}
	if m[1] == "-" {
		return -n
	}
	return n
}

var rubyIntPrefix = regexp.MustCompile(`^[ \t\n\v\f\r]*([+-]?)(\d+(?:_\d+)*)`)

func isoDateOrEmpty(s string) string {
	if !v1DatePattern.MatchString(s) {
		return ""
	}
	if _, err := time.Parse(time.DateOnly, s); err != nil {
		return ""
	}
	return s
}

type v1Order struct {
	ID           int64   `json:"id"`
	OrderNumber  string  `json:"order_number"`
	OrderType    string  `json:"order_type"`
	OrderDate    string  `json:"order_date"`
	CustomerName string  `json:"customer_name"`
	CustomerCode string  `json:"customer_code"`
	ProductName  string  `json:"product_name"`
	ProductCode  string  `json:"product_code"`
	Quantity     int     `json:"quantity"`
	UnitPrice    string  `json:"unit_price"`
	TotalAmount  string  `json:"total_amount"`
	Status       string  `json:"status"`
	DeliveryDate *string `json:"delivery_date"`
	Notes        string  `json:"notes"`
	CreatedAt    string  `json:"created_at"`
	UpdatedAt    string  `json:"updated_at"`
}

type v1Summary struct {
	OrderCount  int      `json:"order_count"`
	Quantity    int      `json:"quantity"`
	TotalAmount string   `json:"total_amount"`
	Statuses    []string `json:"statuses"`
	cents       int64
}

type v1TreeNode struct {
	ID       string       `json:"id"`
	Kind     string       `json:"kind"`
	Depth    int          `json:"depth"`
	Label    string       `json:"label"`
	Order    *v1Order     `json:"order,omitempty"`
	Summary  v1Summary    `json:"summary"`
	Children []v1TreeNode `json:"children"`
}

type v1Meta struct {
	Total      int64 `json:"total"`
	Page       int   `json:"page"`
	PerPage    int   `json:"per_page"`
	TotalPages int   `json:"total_pages"`
}

// toCents converts a NUMERIC(…, 2) value scanned as float64 back to exact cents.
func toCents(v float64) int64 { return int64(math.Round(v * 100)) }

// formatCents renders cents the way Rails renders BigDecimal in JSON
// (BigDecimal#to_s("F")): no trailing zeros, but at least one fraction digit.
func formatCents(cents int64) string {
	sign := ""
	if cents < 0 {
		sign, cents = "-", -cents
	}
	whole, frac := cents/100, cents%100
	fraction := strconv.FormatInt(frac/10, 10)
	if frac%10 != 0 {
		fraction += strconv.FormatInt(frac%10, 10)
	}
	return sign + strconv.FormatInt(whole, 10) + "." + fraction
}

// dateOnly keeps YYYY-MM-DD from the RFC 3339 text database/sql produces when
// it scans a DATE into a string.
func dateOnly(s string) string {
	if len(s) < len(time.DateOnly) {
		return s
	}
	return s[:len(time.DateOnly)]
}

func optionalDateOnly(s *string) *string {
	if s == nil {
		return nil
	}
	d := dateOnly(*s)
	return &d
}

// railsTimestamp matches ActiveSupport's JSON encoding of a UTC time.
func railsTimestamp(t time.Time) string { return t.UTC().Format("2006-01-02T15:04:05.000Z") }

func v1OrderFrom(o model.Order) v1Order {
	return v1Order{
		ID: o.ID, OrderNumber: o.OrderNumber, OrderType: o.OrderType, OrderDate: dateOnly(o.OrderDate),
		CustomerName: o.CustomerName, CustomerCode: o.CustomerCode,
		ProductName: o.ProductName, ProductCode: o.ProductCode, Quantity: o.Quantity,
		UnitPrice: formatCents(toCents(o.UnitPrice)), TotalAmount: formatCents(toCents(o.TotalAmount)),
		Status: o.Status, DeliveryDate: optionalDateOnly(o.DeliveryDate), Notes: o.Notes,
		CreatedAt: railsTimestamp(o.CreatedAt), UpdatedAt: railsTimestamp(o.UpdatedAt),
	}
}

func newBranch(id, kind string, depth int, label string) v1TreeNode {
	return v1TreeNode{ID: id, Kind: kind, Depth: depth, Label: label,
		Summary: v1Summary{Statuses: []string{}}, Children: []v1TreeNode{}}
}

func (s *v1Summary) add(o model.Order, cents int64) {
	s.OrderCount++
	s.Quantity += o.Quantity
	s.cents += cents
	s.TotalAmount = formatCents(s.cents)
	if !slices.Contains(s.Statuses, o.Status) {
		s.Statuses = append(s.Statuses, o.Status)
	}
}

// buildV1Tree is Order.build_tree: customer → product → order, in first-seen
// order, with summaries summed in exact cents.
func buildV1Tree(orders []model.Order) []v1TreeNode {
	customers := []v1TreeNode{}
	customerIdx := map[string]int{}
	productIdx := map[string]int{}

	for _, o := range orders {
		cid := "customer:" + o.CustomerCode
		ci, ok := customerIdx[cid]
		if !ok {
			customers = append(customers, newBranch(cid, "customer", 0, o.CustomerName))
			ci = len(customers) - 1
			customerIdx[cid] = ci
		}
		cents := toCents(o.TotalAmount)
		customer := &customers[ci]
		customer.Summary.add(o, cents)

		pid := cid + ":product:" + o.ProductCode
		pi, ok := productIdx[pid]
		if !ok {
			customer.Children = append(customer.Children, newBranch(pid, "product", 1, o.ProductName))
			pi = len(customer.Children) - 1
			productIdx[pid] = pi
		}
		product := &customer.Children[pi]
		product.Summary.add(o, cents)

		payload := v1OrderFrom(o)
		product.Children = append(product.Children, v1TreeNode{
			ID: "order:" + strconv.FormatInt(o.ID, 10), Kind: "order", Depth: 2, Label: o.OrderNumber, Order: &payload,
			Summary:  v1Summary{OrderCount: 1, Quantity: o.Quantity, TotalAmount: payload.TotalAmount, Statuses: []string{o.Status}},
			Children: []v1TreeNode{},
		})
	}
	return customers
}

func v1MetaFrom(total int64, p model.QueryParams) v1Meta {
	return v1Meta{Total: total, Page: p.Page, PerPage: p.PerPage,
		TotalPages: int(math.Ceil(float64(total) / float64(p.PerPage)))}
}

// v1Handler runs the shared search and lets build shape the body.
func v1Handler(svc service.OrderService, build func([]model.Order, v1Meta) any) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		params := parseV1Params(r.URL.Query())
		resp, err := svc.GetOrders(params)
		if err != nil {
			log.Printf("v1 search failed: %v", err)
			http.Error(w, "Database error", http.StatusInternalServerError)
			return
		}
		w.Header().Set("X-Total-Count", strconv.FormatInt(resp.Total, 10))
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		if err := json.NewEncoder(w).Encode(build(resp.Data, v1MetaFrom(resp.Total, params))); err != nil {
			log.Printf("failed to encode response: %v", err)
		}
	}
}

func V1OrdersHandler(svc service.OrderService) http.HandlerFunc {
	return v1Handler(svc, func(orders []model.Order, meta v1Meta) any {
		payload := make([]v1Order, len(orders))
		for i, o := range orders {
			payload[i] = v1OrderFrom(o)
		}
		return struct {
			Orders []v1Order `json:"orders"`
			Meta   v1Meta    `json:"meta"`
		}{payload, meta}
	})
}

func V1OrderTreeHandler(svc service.OrderService) http.HandlerFunc {
	return v1Handler(svc, func(orders []model.Order, meta v1Meta) any {
		return struct {
			Tree []v1TreeNode `json:"tree"`
			Meta v1Meta       `json:"meta"`
		}{buildV1Tree(orders), meta}
	})
}
