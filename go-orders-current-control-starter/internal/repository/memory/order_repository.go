package memory

import (
	"context"

	"example.com/order-service/internal/order"
)

type OrderRepository struct {
	items  []order.Order
	nextID int64
}

func NewOrderRepository() *OrderRepository {
	return &OrderRepository{
		items:  make([]order.Order, 0),
		nextID: 1,
	}
}

func cloneOrder(value order.Order) order.Order {
	value.Items = append([]order.Item(nil), value.Items...)
	value.History = append([]order.StatusChange(nil), value.History...)
	return value
}

func (r *OrderRepository) Create(_ context.Context, value order.Order) (order.Order, error) {
	value.ID = r.nextID
	r.items = append(r.items, cloneOrder(value))
	r.nextID++
	return value, nil
}

func (r *OrderRepository) GetByID(_ context.Context, id int64) (order.Order, error) {
	if id >= r.nextID || id <= 0 {
		return order.Order{}, order.ErrOrderNotFound
	}
	return cloneOrder(r.items[id-1]), nil
}

func (r *OrderRepository) List(_ context.Context) ([]order.Order, error) {
	orders := make([]order.Order, 0, r.nextID-1)
	for _, item := range r.items {
		orders = append(orders, cloneOrder(item))
	}
	return orders, nil
}

func (r *OrderRepository) Update(_ context.Context, value order.Order) (order.Order, error) {
	if value.ID >= r.nextID || value.ID <= 0 {
		return order.Order{}, order.ErrOrderNotFound
	}
	r.items[value.ID-1] = cloneOrder(value)
	return cloneOrder(value), nil
}
