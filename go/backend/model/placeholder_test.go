package model

import "testing"

func TestDollarPlaceholders(t *testing.T) {
	cases := []struct{ in, want string }{
		{"SELECT 1", "SELECT 1"},
		{"a = ?", "a = $1"},
		{"a = ? AND b LIKE ? LIMIT ? OFFSET ?", "a = $1 AND b LIKE $2 LIMIT $3 OFFSET $4"},
	}
	for _, c := range cases {
		if got := DollarPlaceholders(c.in); got != c.want {
			t.Errorf("DollarPlaceholders(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestDollarPlaceholders_NumbersPastNine(t *testing.T) {
	q, args := BuildQuery(QueryParams{
		Page: 300, PerPage: 50, OrderType: "rush", Status: "shipped",
		CustomerName: "a", ProductName: "b", DateFrom: "2026-01-01", DateTo: "2026-12-31",
	})
	got := DollarPlaceholders(q)
	want := "SELECT o.id, o.order_number, o.order_type, o.order_date, o.customer_name, o.customer_code, " +
		"o.product_name, o.product_code, o.quantity, o.unit_price, o.total_amount, o.status, o.delivery_date, " +
		"o.notes, o.created_at, o.updated_at FROM orders o INNER JOIN (SELECT id FROM orders WHERE order_type = $1 " +
		"AND status = $2 AND customer_name LIKE $3 AND product_name LIKE $4 AND order_date >= $5 AND order_date <= $6 " +
		"ORDER BY id ASC LIMIT $7 OFFSET $8) sub ON o.id = sub.id ORDER BY o.id ASC"
	if got != want {
		t.Errorf("got  %s\nwant %s", got, want)
	}
	if len(args) != 8 {
		t.Errorf("len(args) = %d, want 8", len(args))
	}
}
