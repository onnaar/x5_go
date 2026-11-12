package service_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"example.com/order-service/internal/order"
	"example.com/order-service/internal/service"
	"github.com/stretchr/testify/require"
)

type dummyRepository struct {
	createFn func(context.Context, order.Order) (order.Order, error)
	getFn    func(context.Context, int64) (order.Order, error)
	listFn   func(context.Context) ([]order.Order, error)
	updateFn func(context.Context, order.Order) (order.Order, error)
}

func (m *dummyRepository) Create(ctx context.Context, value order.Order) (order.Order, error) {
	if m.createFn != nil {
		return m.createFn(ctx, value)
	}
	return value, nil
}

func (m *dummyRepository) GetByID(ctx context.Context, id int64) (order.Order, error) {
	if m.getFn != nil {
		return m.getFn(ctx, id)
	}
	return order.Order{}, order.ErrOrderNotFound
}

func (m *dummyRepository) List(ctx context.Context) ([]order.Order, error) {
	if m.listFn != nil {
		return m.listFn(ctx)
	}
	return nil, nil
}

func (m *dummyRepository) Update(ctx context.Context, value order.Order) (order.Order, error) {
	if m.updateFn != nil {
		return m.updateFn(ctx, value)
	}
	return value, nil
}

func TestOrderServiceCreate(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		items     []order.Item
		wantError error
		wantTotal int64
	}{
		{
			name: "valid order",
			items: []order.Item{
				{Name: "keyboard", Quantity: 2, Price: 500},
				{Name: "mouse", Quantity: 1, Price: 250},
			},
			wantTotal: 1250,
		},
		{name: "without items", wantError: order.ErrInvalidOrder},
		{
			name:      "empty name",
			items:     []order.Item{{Name: "  ", Quantity: 1, Price: 100}},
			wantError: order.ErrInvalidOrder,
		},
		{
			name:      "zero quantity",
			items:     []order.Item{{Name: "keyboard", Quantity: 0, Price: 100}},
			wantError: order.ErrInvalidOrder,
		},
		{
			name:      "negative quantity",
			items:     []order.Item{{Name: "keyboard", Quantity: -1, Price: 100}},
			wantError: order.ErrInvalidOrder,
		},
		{
			name:      "zero price",
			items:     []order.Item{{Name: "keyboard", Quantity: 1, Price: 0}},
			wantError: order.ErrInvalidOrder,
		},
		{
			name:      "negative price",
			items:     []order.Item{{Name: "keyboard", Quantity: 1, Price: -100}},
			wantError: order.ErrInvalidOrder,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var saved order.Order
			repository := &dummyRepository{
				createFn: func(_ context.Context, value order.Order) (order.Order, error) {
					saved = value
					value.ID = 1
					return value, nil
				},
			}
			svc := service.NewOrderService(repository)
			got, err := svc.Create(context.Background(), order.CreateOrderInput{Items: tt.items})
			if tt.wantError != nil {
				require.ErrorIs(t, err, tt.wantError)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.wantTotal, saved.TotalPrice)
			require.Equal(t, order.StatusNew, saved.Status)
			require.Len(t, saved.History, 1)
			require.Equal(t, order.StatusNew, saved.History[0].Status)
			require.False(t, saved.CreatedAt.IsZero())
			require.Equal(t, int64(1), got.ID)
		})
	}
}

func TestOrderServiceCreateCopiesItems(t *testing.T) {
	t.Parallel()
	inputItems := []order.Item{{Name: "keyboard", Quantity: 1, Price: 100}}
	var saved order.Order
	repository := &dummyRepository{
		createFn: func(_ context.Context, value order.Order) (order.Order, error) {
			saved = value
			return value, nil
		},
	}
	_, err := service.NewOrderService(repository).Create(context.Background(), order.CreateOrderInput{Items: inputItems})
	require.NoError(t, err)
	inputItems[0].Name = "changed"
	require.Equal(t, "keyboard", saved.Items[0].Name)
}

func TestOrderService_GetByID(t *testing.T) {
	t.Parallel()
	want := order.Order{ID: 7, Status: order.StatusNew}
	repositoryError := errors.New("repository failure")
	tests := []struct {
		name      string
		repoOrder order.Order
		repoError error
		want      order.Order
		wantError error
	}{
		{name: "found", repoOrder: want, want: want},
		{name: "repository error", repoError: repositoryError, wantError: repositoryError},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			repository := &dummyRepository{
				getFn: func(_ context.Context, _ int64) (order.Order, error) {
					return tt.repoOrder, tt.repoError
				},
			}
			got, err := service.NewOrderService(repository).GetByID(context.Background(), 7)
			require.ErrorIs(t, err, tt.wantError)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestOrderServiceList(t *testing.T) {
	t.Parallel()
	want := []order.Order{{ID: 1}, {ID: 2}}
	repositoryError := errors.New("repository failure")
	tests := []struct {
		name      string
		orders    []order.Order
		repoError error
		wantError error
	}{
		{name: "returns orders", orders: want},
		{name: "repository error", repoError: repositoryError, wantError: repositoryError},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			repository := &dummyRepository{
				listFn: func(_ context.Context) ([]order.Order, error) {
					return tt.orders, tt.repoError
				},
			}
			got, err := service.NewOrderService(repository).List(context.Background())
			require.ErrorIs(t, err, tt.wantError)
			require.Equal(t, tt.orders, got)
		})
	}
}

func TestOrderServiceUpdateStatus(t *testing.T) {
	t.Parallel()
	repositoryError := errors.New("repository failure")
	tests := []struct {
		name          string
		currentStatus order.Status
		newStatus     order.Status
		getError      error
		updateError   error
		wantError     error
	}{
		{
			name:          "new to processing",
			currentStatus: order.StatusNew,
			newStatus:     order.StatusProcessing,
		},
		{
			name:          "new to cancelled",
			currentStatus: order.StatusNew,
			newStatus:     order.StatusCancelled,
		},
		{
			name:          "processing to completed",
			currentStatus: order.StatusProcessing,
			newStatus:     order.StatusCompleted,
		},
		{
			name:          "processing to cancelled",
			currentStatus: order.StatusProcessing,
			newStatus:     order.StatusCancelled,
		},
		{
			name:          "invalid target status",
			currentStatus: order.StatusNew,
			newStatus:     "unknown",
			wantError:     order.ErrInvalidStatus,
		},
		{
			name:          "new to completed",
			currentStatus: order.StatusNew,
			newStatus:     order.StatusCompleted,
			wantError:     order.ErrInvalidStatusTransition,
		},
		{
			name:          "processing to new",
			currentStatus: order.StatusProcessing,
			newStatus:     order.StatusNew,
			wantError:     order.ErrInvalidStatusTransition,
		},
		{
			name:          "completed is final",
			currentStatus: order.StatusCompleted,
			newStatus:     order.StatusProcessing,
			wantError:     order.ErrInvalidStatusTransition,
		},
		{
			name:          "cancelled is final",
			currentStatus: order.StatusCancelled,
			newStatus:     order.StatusProcessing,
			wantError:     order.ErrInvalidStatusTransition,
		},
		{
			name:      "get error",
			getError:  repositoryError,
			newStatus: order.StatusProcessing,
			wantError: repositoryError,
		},
		{
			name:          "update error",
			currentStatus: order.StatusNew,
			newStatus:     order.StatusProcessing,
			updateError:   repositoryError,
			wantError:     repositoryError,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var updated order.Order
			repository := &dummyRepository{
				getFn: func(_ context.Context, _ int64) (order.Order, error) {
					return order.Order{
						ID:      1,
						Status:  tt.currentStatus,
						History: []order.StatusChange{{Status: tt.currentStatus, ChangedAt: time.Unix(1, 0)}},
					}, tt.getError
				},
				updateFn: func(_ context.Context, value order.Order) (order.Order, error) {
					updated = value
					return value, tt.updateError
				},
			}
			got, err := service.NewOrderService(repository).UpdateStatus(context.Background(), 1, tt.newStatus)
			require.ErrorIs(t, err, tt.wantError)
			if tt.wantError == nil {
				require.NoError(t, err)
				require.Equal(t, tt.newStatus, updated.Status)
				require.Len(t, updated.History, 2)
				require.Equal(t, tt.newStatus, updated.History[1].Status)
				require.Equal(t, updated, got)
			}
		})
	}
}

func TestOrderServiceHistory(t *testing.T) {
	t.Parallel()
	repositoryError := errors.New("repository failure")
	want := []order.StatusChange{
		{Status: order.StatusNew, ChangedAt: time.Unix(1, 0)},
		{Status: order.StatusProcessing, ChangedAt: time.Unix(2, 0)},
	}
	tests := []struct {
		name      string
		history   []order.StatusChange
		repoError error
		wantError error
	}{
		{name: "chronological history", history: want},
		{name: "repository error", repoError: repositoryError, wantError: repositoryError},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			repository := &dummyRepository{
				getFn: func(_ context.Context, _ int64) (order.Order, error) {
					return order.Order{History: tt.history}, tt.repoError
				},
			}
			got, err := service.NewOrderService(repository).History(context.Background(), 1)
			require.ErrorIs(t, err, tt.wantError)
			if tt.wantError == nil {
				require.Equal(t, tt.history, got)
				got[0].Status = order.StatusCancelled
				require.Equal(t, order.StatusNew, tt.history[0].Status)
			}
		})
	}
}
