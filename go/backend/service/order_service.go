package service

import (
	"math"

	"bigtable-backend/model"
	"bigtable-backend/repository"
)

// OrderService defines the interface for order business logic
type OrderService interface {
	GetOrders(params model.QueryParams) (model.OrdersResponse, error)
}

type orderService struct {
	repo repository.OrderRepository
}

// NewOrderService creates a new OrderService
func NewOrderService(repo repository.OrderRepository) OrderService {
	return &orderService{repo: repo}
}

func (s *orderService) GetOrders(params model.QueryParams) (model.OrdersResponse, error) {
	total, err := s.repo.CountOrders(params)
	if err != nil {
		return model.OrdersResponse{}, err
	}

	orders, err := s.repo.FindOrders(params)
	if err != nil {
		return model.OrdersResponse{}, err
	}

	totalPages := int(math.Ceil(float64(total) / float64(params.PerPage)))

	return model.OrdersResponse{
		Data:       orders,
		Total:      total,
		Page:       params.Page,
		PerPage:    params.PerPage,
		TotalPages: totalPages,
	}, nil
}
