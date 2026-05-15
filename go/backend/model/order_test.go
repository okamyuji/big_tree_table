package model

import (
	"strings"
	"testing"
)

func TestQueryParams_Normalize(t *testing.T) {
	cases := []struct {
		name string
		in   QueryParams
		want QueryParams
	}{
		{
			name: "defaults applied to empty struct",
			in:   QueryParams{},
			want: QueryParams{Page: 1, PerPage: 1, Sort: "id", Order: "asc"},
		},
		{
			name: "page below 1 clamped up",
			in:   QueryParams{Page: -5, PerPage: 50, Sort: "id", Order: "asc"},
			want: QueryParams{Page: 1, PerPage: 50, Sort: "id", Order: "asc"},
		},
		{
			name: "page above MaxPage clamped down",
			in:   QueryParams{Page: MaxPage + 1, PerPage: 50, Sort: "id", Order: "asc"},
			want: QueryParams{Page: MaxPage, PerPage: 50, Sort: "id", Order: "asc"},
		},
		{
			name: "per_page above MaxPerPage clamped down",
			in:   QueryParams{Page: 1, PerPage: 999, Sort: "id", Order: "asc"},
			want: QueryParams{Page: 1, PerPage: MaxPerPage, Sort: "id", Order: "asc"},
		},
		{
			name: "valid values preserved",
			in:   QueryParams{Page: 5, PerPage: 25, Sort: "order_date", Order: "desc"},
			want: QueryParams{Page: 5, PerPage: 25, Sort: "order_date", Order: "desc"},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := c.in
			got.Normalize()
			if got != c.want {
				t.Errorf("Normalize() = %#v, want %#v", got, c.want)
			}
		})
	}
}

func TestBuildQuery_NoFilters(t *testing.T) {
	p := QueryParams{Page: 1, PerPage: 50, Sort: "id", Order: "asc"}
	query, args := BuildQuery(p)
	if len(args) != 0 {
		t.Errorf("expected 0 args, got %d", len(args))
	}
	if strings.Contains(query, "WHERE") {
		t.Error("expected no WHERE clause")
	}
	if !strings.Contains(query, "ORDER BY id ASC") {
		t.Errorf("expected ORDER BY id ASC, got %s", query)
	}
	if !strings.Contains(query, "LIMIT 50 OFFSET 0") {
		t.Errorf("expected LIMIT 50 OFFSET 0, got %s", query)
	}
}

func TestBuildQuery_WithStatusFilter(t *testing.T) {
	p := QueryParams{Page: 1, PerPage: 50, Sort: "id", Order: "asc", Status: "受注確認"}
	query, args := BuildQuery(p)
	if len(args) != 1 {
		t.Errorf("expected 1 arg, got %d", len(args))
	}
	if args[0] != "受注確認" {
		t.Errorf("expected arg '受注確認', got %v", args[0])
	}
	if !strings.Contains(query, "WHERE status = ?") {
		t.Errorf("expected WHERE status = ?, got %s", query)
	}
}

func TestBuildQuery_WithDateRange(t *testing.T) {
	p := QueryParams{Page: 1, PerPage: 50, Sort: "id", Order: "asc", DateFrom: "2023-01-01", DateTo: "2023-12-31"}
	query, args := BuildQuery(p)
	if len(args) != 2 {
		t.Errorf("expected 2 args, got %d", len(args))
	}
	if !strings.Contains(query, "order_date >= ?") {
		t.Errorf("expected order_date >= ?, got %s", query)
	}
	if !strings.Contains(query, "order_date <= ?") {
		t.Errorf("expected order_date <= ?, got %s", query)
	}
}

func TestBuildQuery_SQLInjectionInSort(t *testing.T) {
	p := QueryParams{Page: 1, PerPage: 50, Sort: "id; DROP TABLE orders;--", Order: "asc"}
	query, _ := BuildQuery(p)
	if !strings.Contains(query, "ORDER BY id ASC") {
		t.Errorf("expected fallback to ORDER BY id ASC, got %s", query)
	}
}

func TestBuildQuery_AllFilters(t *testing.T) {
	p := QueryParams{
		Page: 2, PerPage: 20, Sort: "order_date", Order: "desc",
		OrderType: "受注", Status: "出荷済み",
		CustomerName: "田中", ProductName: "ボルト",
		DateFrom: "2023-01-01", DateTo: "2023-12-31",
	}
	query, args := BuildQuery(p)
	if len(args) != 6 {
		t.Errorf("expected 6 args, got %d", len(args))
	}
	if !strings.Contains(query, "ORDER BY order_date DESC") {
		t.Errorf("expected ORDER BY order_date DESC, got %s", query)
	}
	if !strings.Contains(query, "LIMIT 20 OFFSET 20") {
		t.Errorf("expected LIMIT 20 OFFSET 20, got %s", query)
	}
	if !strings.Contains(query, "customer_name LIKE ?") {
		t.Error("expected customer_name LIKE ?")
	}
	// Check LIKE args have wildcards
	if args[2] != "%田中%" {
		t.Errorf("expected %%田中%%, got %v", args[2])
	}
}

func TestBuildQuery_LikeWildcardEscaped(t *testing.T) {
	p := QueryParams{Page: 1, PerPage: 50, Sort: "id", Order: "asc", CustomerName: "100%株式会社"}
	_, args := BuildQuery(p)
	if len(args) != 1 {
		t.Fatalf("expected 1 arg, got %d", len(args))
	}
	got := args[0].(string)
	expected := `%100\%株式会社%`
	if got != expected {
		t.Errorf("expected LIKE arg %q, got %q", expected, got)
	}
}

func TestBuildQuery_InvalidDateIgnored(t *testing.T) {
	p := QueryParams{Page: 1, PerPage: 50, Sort: "id", Order: "asc", DateFrom: "not-a-date", DateTo: "2024-01-01"}
	query, args := BuildQuery(p)
	// 不正なDateFromは無視され、DateToだけが条件に含まれる
	if len(args) != 1 {
		t.Fatalf("expected 1 arg (only valid date_to), got %d", len(args))
	}
	if !strings.Contains(query, "order_date <= ?") {
		t.Errorf("expected order_date <= ?, got %s", query)
	}
	if strings.Contains(query, "order_date >= ?") {
		t.Errorf("invalid date_from should be ignored, got %s", query)
	}
}

func TestBuildQuery_DeferredJoinForLargeOffset(t *testing.T) {
	// OFFSETが閾値以上の場合、deferred joinに切り替わることを確認
	p := QueryParams{Page: 501, PerPage: 50, Sort: "order_date", Order: "desc"}
	query, _ := BuildQuery(p)
	if !strings.Contains(query, "INNER JOIN") {
		t.Errorf("expected deferred join with INNER JOIN for large offset, got %s", query)
	}
	if !strings.Contains(query, "SELECT id FROM orders") {
		t.Errorf("expected subquery to select only id, got %s", query)
	}
	if !strings.Contains(query, "LIMIT 50 OFFSET 25000") {
		t.Errorf("expected LIMIT 50 OFFSET 25000, got %s", query)
	}
}

func TestBuildQuery_SmallOffsetNoDeferred(t *testing.T) {
	// OFFSETが閾値未満の場合、通常のクエリのままであることを確認
	p := QueryParams{Page: 10, PerPage: 50, Sort: "id", Order: "asc"}
	query, _ := BuildQuery(p)
	if strings.Contains(query, "INNER JOIN") {
		t.Errorf("expected simple query for small offset, got %s", query)
	}
}

func TestBuildCountQuery_NoFilters(t *testing.T) {
	p := QueryParams{Page: 1, PerPage: 50}
	query, args := BuildCountQuery(p)
	if len(args) != 0 {
		t.Errorf("expected 0 args, got %d", len(args))
	}
	if query != "SELECT COUNT(*) FROM orders " {
		t.Errorf("unexpected query: %s", query)
	}
}

func TestBuildCountQuery_WithFilters(t *testing.T) {
	p := QueryParams{Page: 1, PerPage: 50, Status: "納品完了"}
	query, args := BuildCountQuery(p)
	if len(args) != 1 {
		t.Errorf("expected 1 arg, got %d", len(args))
	}
	if !strings.Contains(query, "WHERE status = ?") {
		t.Errorf("expected WHERE status = ?, got %s", query)
	}
}

func TestBuildOrderTree(t *testing.T) {
	orders := []Order{
		{
			ID: 1, OrderNumber: "ORD-001", CustomerCode: "C001", CustomerName: "顧客A",
			ProductCode: "P001", ProductName: "商品A", Quantity: 2, TotalAmount: 3000, Status: "受注確認",
		},
		{
			ID: 2, OrderNumber: "ORD-002", CustomerCode: "C001", CustomerName: "顧客A",
			ProductCode: "P001", ProductName: "商品A", Quantity: 3, TotalAmount: 4500, Status: "出荷済み",
		},
		{
			ID: 3, OrderNumber: "ORD-003", CustomerCode: "C001", CustomerName: "顧客A",
			ProductCode: "P002", ProductName: "商品B", Quantity: 5, TotalAmount: 8000, Status: "受注確認",
		},
	}

	tree := BuildOrderTree(orders)
	if len(tree) != 1 {
		t.Fatalf("expected 1 customer node, got %d", len(tree))
	}
	customer := tree[0]
	if customer.ID != "customer:C001" {
		t.Errorf("unexpected customer id: %s", customer.ID)
	}
	if customer.Summary.OrderCount != 3 {
		t.Errorf("expected customer order count 3, got %d", customer.Summary.OrderCount)
	}
	if customer.Summary.Quantity != 10 {
		t.Errorf("expected quantity 10, got %d", customer.Summary.Quantity)
	}
	if len(customer.Children) != 2 {
		t.Fatalf("expected 2 product nodes, got %d", len(customer.Children))
	}
	if len(customer.Children[0].Children) != 2 {
		t.Errorf("expected first product to have 2 order children, got %d", len(customer.Children[0].Children))
	}
	if customer.Children[0].Children[0].Order == nil {
		t.Fatal("expected order leaf to include order payload")
	}
	if customer.Children[0].Children[0].Order.OrderNumber != "ORD-001" {
		t.Errorf("unexpected order number: %s", customer.Children[0].Children[0].Order.OrderNumber)
	}
}
