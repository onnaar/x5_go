package repository

import (
	"errors"
	"sort"
	"sync"

	"example.com/lesson9practice/internal/model"
)

var ErrNotFound = errors.New("order not found")
var ErrAlreadyCancelled = errors.New("order already cancelled")

type Memory struct {
	mu     sync.Mutex
	nextID int
	orders map[int]model.Order
}

func New() *Memory { return &Memory{nextID: 1, orders: make(map[int]model.Order)} }

func (m *Memory) Create(customer string) (model.Order, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	order := model.Order{ID: m.nextID, Customer: customer, Status: model.StatusNew}
	m.orders[order.ID] = order
	m.nextID++
	return order, nil
}

func (m *Memory) List(limit, offset int, status string) ([]model.Order, error) {
	m.mu.Lock()
	orders := make([]model.Order, 0, len(m.orders))
	for _, order := range m.orders {
		if status == "" || order.Status == status {
			orders = append(orders, order)
		}
	}
	m.mu.Unlock()

	if offset >= len(orders) {
		return orders[:0], nil
	}
	sort.Slice(orders, func(i, j int) bool { return orders[i].ID < orders[j].ID })
	page := orders[offset:]
	return page[:min(limit, len(page))], nil
}

func (m *Memory) Cancel(id int) (model.Order, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	order, ok := m.orders[id]
	if !ok {
		return model.Order{}, ErrNotFound
	}
	if order.Status != model.StatusNew {
		return model.Order{}, ErrAlreadyCancelled
	}
	order.Status = model.StatusCancelled
	m.orders[id] = order
	return order, nil
}
func (m *Memory) ByID(id int) (model.Order, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	order, ok := m.orders[id]
	if !ok {
		return model.Order{}, ErrNotFound
	}
	return order, nil
}
