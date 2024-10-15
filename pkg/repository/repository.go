package repository

import "database/sql"

type Repoitory struct {
	Product *ProductRepository
	Order   *OrderRepository
}

func NewRepository(db *sql.DB) *Repoitory {
	return &Repoitory{
		Product: NewProductRepository(db),
		Order: NewOrderRepository(db),
	}
}