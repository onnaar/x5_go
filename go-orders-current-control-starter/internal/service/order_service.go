package service

import (
	"context"
	"strings"
	"time"

	"example.com/order-service/internal/order"
)

type OrderService struct {
	repository order.Repository
}

func NewOrderService(repository order.Repository) *OrderService {
	return &OrderService{repository: repository}
}

func (s *OrderService) Create(ctx context.Context, input order.CreateOrderInput) (order.Order, error) {
	if len(input.Items) == 0 {
		return order.Order{}, order.ErrInvalidOrder
	}
	var total int64
	items := make([]order.Item, len(input.Items))
	copy(items, input.Items)
	for _, item := range input.Items {
		if strings.TrimSpace(item.Name) == "" || item.Quantity <= 0 || item.Price <= 0 {
			return order.Order{}, order.ErrInvalidOrder
		}
		total += int64(item.Quantity) * item.Price
	}
	now := time.Now()
	newOrder := order.Order{
		Items:      items,
		TotalPrice: total,
		Status:     order.StatusNew,
		CreatedAt:  now,
		History: []order.StatusChange{
			{
				Status:    order.StatusNew,
				ChangedAt: now,
			},
		},
	}
	return s.repository.Create(ctx, newOrder)
}

func (s *OrderService) GetByID(ctx context.Context, id int64) (order.Order, error) {
	return s.repository.GetByID(ctx, id)
}

func (s *OrderService) List(ctx context.Context) ([]order.Order, error) {
	return s.repository.List(ctx)
}

func (s *OrderService) UpdateStatus(ctx context.Context, id int64, status order.Status) (order.Order, error) {
	switch status {
	case order.StatusNew, order.StatusProcessing, order.StatusCompleted, order.StatusCancelled:
	default:
		return order.Order{}, order.ErrInvalidStatus
	}
	curOrder, err := s.repository.GetByID(ctx, id)
	if err != nil {
		return order.Order{}, err
	}
	ordStatus := curOrder.Status
	if !(ordStatus == order.StatusNew && (status == order.StatusProcessing || status == order.StatusCancelled) ||
		ordStatus == order.StatusProcessing && (status == order.StatusCompleted || status == order.StatusCancelled)) {
		return order.Order{}, order.ErrInvalidStatusTransition
	}
	curOrder.Status = status
	curOrder.History = append(curOrder.History, order.StatusChange{
		Status:    status,
		ChangedAt: time.Now(),
	})
	return s.repository.Update(ctx, curOrder)
}

func (s *OrderService) History(ctx context.Context, id int64) ([]order.StatusChange, error) {
	curOrder, err := s.repository.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	changes := make([]order.StatusChange, len(curOrder.History))
	copy(changes, curOrder.History)
	return changes, nil
}
