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

// SQLOrderRepository implements OrderRepository over MySQL or PostgreSQL.
type SQLOrderRepository struct {
	db       *sql.DB
	postgres bool
}

// NewMySQLOrderRepository creates a repository that sends the `?` placeholders as they are.
func NewMySQLOrderRepository(db *sql.DB) *SQLOrderRepository {
	return &SQLOrderRepository{db: db}
}

// NewPostgresOrderRepository creates a repository that rewrites placeholders to $1, $2, ….
func NewPostgresOrderRepository(db *sql.DB) *SQLOrderRepository {
	return &SQLOrderRepository{db: db, postgres: true}
}

func (r *SQLOrderRepository) rebind(query string) string {
	if r.postgres {
		return model.DollarPlaceholders(query)
	}
	return query
}

func (r *SQLOrderRepository) FindOrders(params model.QueryParams) ([]model.Order, error) {
	query, args := model.BuildQuery(params)
	rows, err := r.db.Query(r.rebind(query), args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	orders := make([]model.Order, 0)
	for rows.Next() {
		var o model.Order
		if err := rows.Scan(o.ScanTargets()...); err != nil {
			return nil, err
		}
		orders = append(orders, o)
	}
	return orders, rows.Err()
}

func (r *SQLOrderRepository) CountOrders(params model.QueryParams) (int64, error) {
	query, args := model.BuildCountQuery(params)
	var total int64
	if err := r.db.QueryRow(r.rebind(query), args...).Scan(&total); err != nil {
		return 0, err
	}
	return total, nil
}
