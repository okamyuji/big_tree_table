package handler

import (
	"net/http"
	"testing"
)

func TestParseQueryParams_Defaults(t *testing.T) {
	r, _ := http.NewRequest("GET", "/api/orders", nil)
	p := parseQueryParams(r)
	if p.Page != 1 {
		t.Errorf("expected page=1, got %d", p.Page)
	}
	if p.PerPage != 50 {
		t.Errorf("expected per_page=50, got %d", p.PerPage)
	}
	if p.Sort != "id" {
		t.Errorf("expected sort=id, got %s", p.Sort)
	}
	if p.Order != "asc" {
		t.Errorf("expected order=asc, got %s", p.Order)
	}
}

func TestParseQueryParams_CustomValues(t *testing.T) {
	r, _ := http.NewRequest("GET", "/api/orders?page=3&per_page=25&sort=order_date&order=desc&status=出荷済み&customer_name=田中", nil)
	p := parseQueryParams(r)
	if p.Page != 3 {
		t.Errorf("expected page=3, got %d", p.Page)
	}
	if p.PerPage != 25 {
		t.Errorf("expected per_page=25, got %d", p.PerPage)
	}
	if p.Sort != "order_date" {
		t.Errorf("expected sort=order_date, got %s", p.Sort)
	}
	if p.Order != "desc" {
		t.Errorf("expected order=desc, got %s", p.Order)
	}
	if p.Status != "出荷済み" {
		t.Errorf("expected status=出荷済み, got %s", p.Status)
	}
	if p.CustomerName != "田中" {
		t.Errorf("expected customer_name=田中, got %s", p.CustomerName)
	}
}

func TestParseQueryParams_InvalidBounds(t *testing.T) {
	r, _ := http.NewRequest("GET", "/api/orders?page=-5&per_page=999", nil)
	p := parseQueryParams(r)
	if p.Page != 1 {
		t.Errorf("expected page clamped to 1, got %d", p.Page)
	}
	if p.PerPage != 100 {
		t.Errorf("expected per_page clamped to 100, got %d", p.PerPage)
	}
}

func TestParseQueryParams_ZeroPerPage(t *testing.T) {
	r, _ := http.NewRequest("GET", "/api/orders?per_page=0", nil)
	p := parseQueryParams(r)
	if p.PerPage != 1 {
		t.Errorf("expected per_page clamped to 1, got %d", p.PerPage)
	}
}
