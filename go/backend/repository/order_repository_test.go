package repository

import (
	"testing"

	"bigtable-backend/model"
)

// mockOrderRepository is a mock implementation of OrderRepository for testing
type mockOrderRepository struct {
	findOrdersFn  func(params model.QueryParams) ([]model.Order, error)
	countOrdersFn func(params model.QueryParams) (int64, error)
}

func (m *mockOrderRepository) FindOrders(params model.QueryParams) ([]model.Order, error) {
	return m.findOrdersFn(params)
}

func (m *mockOrderRepository) CountOrders(params model.QueryParams) (int64, error) {
	return m.countOrdersFn(params)
}

func TestOrderRepositoryInterface(t *testing.T) {
	// Verify that MySQLOrderRepository implements OrderRepository
	var _ OrderRepository = &MySQLOrderRepository{}

	// Verify that the mock also implements the interface
	var _ OrderRepository = &mockOrderRepository{}
}

func TestMockOrderRepository_FindOrders(t *testing.T) {
	expected := []model.Order{{ID: 1, OrderNumber: "ORD-001"}}
	mock := &mockOrderRepository{
		findOrdersFn: func(params model.QueryParams) ([]model.Order, error) {
			return expected, nil
		},
	}

	params := model.QueryParams{Page: 1, PerPage: 50, Sort: "id", Order: "asc"}
	orders, err := mock.FindOrders(params)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(orders) != 1 {
		t.Errorf("expected 1 order, got %d", len(orders))
	}
	if orders[0].OrderNumber != "ORD-001" {
		t.Errorf("expected ORD-001, got %s", orders[0].OrderNumber)
	}
}

func TestMockOrderRepository_CountOrders(t *testing.T) {
	mock := &mockOrderRepository{
		countOrdersFn: func(params model.QueryParams) (int64, error) {
			return 42, nil
		},
	}

	params := model.QueryParams{Page: 1, PerPage: 50}
	count, err := mock.CountOrders(params)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if count != 42 {
		t.Errorf("expected 42, got %d", count)
	}
}
