package model

import (
	"strings"
	"testing"
)

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
