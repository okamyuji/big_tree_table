package repository

import (
	"database/sql"

	"bigtable-backend/model"
)

// OrderRepository defines the interface for order data access
type OrderRepository interface {
	FindOrders(params model.QueryParams) ([]model.Order, error)
	CountOrders(params model.QueryParams) (int64, error)
}

// MySQLOrderRepository implements OrderRepository using MySQL
type MySQLOrderRepository struct {
	db *sql.DB
}

// NewMySQLOrderRepository creates a new MySQLOrderRepository
func NewMySQLOrderRepository(db *sql.DB) *MySQLOrderRepository {
	return &MySQLOrderRepository{db: db}
}

func (r *MySQLOrderRepository) FindOrders(params model.QueryParams) ([]model.Order, error) {
	query, args := model.BuildQuery(params)
	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	orders := make([]model.Order, 0)
	for rows.Next() {
		var o model.Order
		if err := rows.Scan(
			&o.ID, &o.OrderNumber, &o.OrderType, &o.OrderDate,
			&o.CustomerName, &o.CustomerCode, &o.ProductName, &o.ProductCode,
			&o.Quantity, &o.UnitPrice, &o.TotalAmount, &o.Status,
			&o.DeliveryDate, &o.Notes, &o.CreatedAt, &o.UpdatedAt,
		); err != nil {
			return nil, err
		}
		orders = append(orders, o)
	}
	return orders, rows.Err()
}

func (r *MySQLOrderRepository) CountOrders(params model.QueryParams) (int64, error) {
	query, args := model.BuildCountQuery(params)
	var total int64
	if err := r.db.QueryRow(query, args...).Scan(&total); err != nil {
		return 0, err
	}
	return total, nil
}
