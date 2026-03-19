package service

import (
	"errors"
	"testing"

	"bigtable-backend/model"
	"bigtable-backend/repository"
)

// mockRepo implements repository.OrderRepository for testing
type mockRepo struct {
	findOrdersFn  func(params model.QueryParams) ([]model.Order, error)
	countOrdersFn func(params model.QueryParams) (int64, error)
}

func (m *mockRepo) FindOrders(params model.QueryParams) ([]model.Order, error) {
	return m.findOrdersFn(params)
}

func (m *mockRepo) CountOrders(params model.QueryParams) (int64, error) {
	return m.countOrdersFn(params)
}

// Verify interface compliance
var _ repository.OrderRepository = &mockRepo{}

func TestGetOrders_Success(t *testing.T) {
	mock := &mockRepo{
		countOrdersFn: func(params model.QueryParams) (int64, error) {
			return 100, nil
		},
		findOrdersFn: func(params model.QueryParams) ([]model.Order, error) {
			return []model.Order{
				{ID: 1, OrderNumber: "ORD-001"},
				{ID: 2, OrderNumber: "ORD-002"},
			}, nil
		},
	}

	svc := NewOrderService(mock)
	params := model.QueryParams{Page: 1, PerPage: 50, Sort: "id", Order: "asc"}

	resp, err := svc.GetOrders(params)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Total != 100 {
		t.Errorf("expected total=100, got %d", resp.Total)
	}
	if len(resp.Data) != 2 {
		t.Errorf("expected 2 orders, got %d", len(resp.Data))
	}
	if resp.Page != 1 {
		t.Errorf("expected page=1, got %d", resp.Page)
	}
	if resp.PerPage != 50 {
		t.Errorf("expected per_page=50, got %d", resp.PerPage)
	}
	if resp.TotalPages != 2 {
		t.Errorf("expected total_pages=2, got %d", resp.TotalPages)
	}
}

func TestGetOrders_CountError(t *testing.T) {
	mock := &mockRepo{
		countOrdersFn: func(params model.QueryParams) (int64, error) {
			return 0, errors.New("db error")
		},
		findOrdersFn: func(params model.QueryParams) ([]model.Order, error) {
			return nil, nil
		},
	}

	svc := NewOrderService(mock)
	params := model.QueryParams{Page: 1, PerPage: 50, Sort: "id", Order: "asc"}

	_, err := svc.GetOrders(params)
	if err == nil {
		t.Error("expected error, got nil")
	}
}

func TestGetOrders_FindError(t *testing.T) {
	mock := &mockRepo{
		countOrdersFn: func(params model.QueryParams) (int64, error) {
			return 10, nil
		},
		findOrdersFn: func(params model.QueryParams) ([]model.Order, error) {
			return nil, errors.New("query error")
		},
	}

	svc := NewOrderService(mock)
	params := model.QueryParams{Page: 1, PerPage: 50, Sort: "id", Order: "asc"}

	_, err := svc.GetOrders(params)
	if err == nil {
		t.Error("expected error, got nil")
	}
}

func TestGetOrders_TotalPages(t *testing.T) {
	tests := []struct {
		name          string
		total         int64
		perPage       int
		expectedPages int
	}{
		{"exact division", 100, 50, 2},
		{"with remainder", 101, 50, 3},
		{"single page", 5, 50, 1},
		{"zero results", 0, 50, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockRepo{
				countOrdersFn: func(params model.QueryParams) (int64, error) {
					return tt.total, nil
				},
				findOrdersFn: func(params model.QueryParams) ([]model.Order, error) {
					return []model.Order{}, nil
				},
			}

			svc := NewOrderService(mock)
			resp, err := svc.GetOrders(model.QueryParams{Page: 1, PerPage: tt.perPage, Sort: "id", Order: "asc"})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if resp.TotalPages != tt.expectedPages {
				t.Errorf("expected %d pages, got %d", tt.expectedPages, resp.TotalPages)
			}
		})
	}
}
