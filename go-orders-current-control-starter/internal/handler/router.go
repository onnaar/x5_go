package handler

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"example.com/order-service/internal/order"
	"github.com/go-chi/chi/v5"
	httpSwagger "github.com/swaggo/http-swagger"
)

type OrderService interface {
	Create(ctx context.Context, input order.CreateOrderInput) (order.Order, error)
	GetByID(ctx context.Context, id int64) (order.Order, error)
	List(ctx context.Context) ([]order.Order, error)
	UpdateStatus(ctx context.Context, id int64, status order.Status) (order.Order, error)
	History(ctx context.Context, id int64) ([]order.StatusChange, error)
}

type Handler struct {
	service OrderService
	logger  *slog.Logger
}

func NewRouter(service OrderService, logger *slog.Logger) http.Handler {
	h := &Handler{service: service, logger: logger}
	router := chi.NewRouter()
	router.Use(recoveryMiddleware(logger))
	router.Use(requestLoggingMiddleware(logger))
	router.Get("/health", h.health)
	router.Get("/swagger/*", httpSwagger.Handler())
	router.Route("/api/v1/orders", func(r chi.Router) {
		r.Post("/", h.createOrder)
		r.Get("/", h.listOrders)
		r.Get("/{id}", h.getOrder)
		r.Patch("/{id}/status", h.updateOrderStatus)
		r.Get("/{id}/history", h.getOrderHistory)
	})

	return router
}

// @Summary проверка сервиса
// @Tags system
// @Success 200
// @Router /health [get]
func (h *Handler) health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// @Summary создать заказ
// @Tags orders
// @Accept json
// @Produce json
// @Param body body order.CreateOrderInput true "данные заказа"
// @Success 201
// @Failure 400
// @Failure 500
// @Router /api/v1/orders [post]
func (h *Handler) createOrder(w http.ResponseWriter, r *http.Request) {
	var input order.CreateOrderInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{
			Error: "invalid json",
			Code:  "INVALID_REQUEST",
		})
		return
	}
	createdOrder, err := h.service.Create(r.Context(), input)
	if err != nil {
		if errors.Is(err, order.ErrInvalidOrder) {
			writeJSON(w, http.StatusBadRequest, errorResponse{
				Error: "invalid order",
				Code:  "INVALID_ORDER",
			})
			return
		}
		h.logger.Error("failed to create order", slog.String("error", err.Error()))
		writeJSON(w, http.StatusInternalServerError, errorResponse{
			Error: "internal server error",
			Code:  "INTERNAL_ERROR",
		})
		return
	}
	writeJSON(w, http.StatusCreated, createdOrder)
}

// @Summary получить список заказов
// @Tags orders
// @Produce json
// @Success 200 {array} order.Order
// @Failure 500 {object} errorResponse
// @Router /api/v1/orders [get]
func (h *Handler) listOrders(w http.ResponseWriter, r *http.Request) {
	orders, err := h.service.List(r.Context())
	if err != nil {
		h.writeInternalError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, orders)
}

// @Summary получить заказ
// @Tags orders
// @Produce json
// @Param id path int64 true "идентификатор заказа"
// @Success 200 {object} order.Order
// @Failure 400 {object} errorResponse
// @Failure 404 {object} errorResponse
// @Router /api/v1/orders/{id} [get]
func (h *Handler) getOrder(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{
			Error: "invalid order id",
			Code:  "INVALID_ID",
		})
		return
	}
	currentOrder, err := h.service.GetByID(r.Context(), id)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, currentOrder)
}

// @Summary изменить статус заказа
// @Tags orders
// @Accept json
// @Produce json
// @Param id path int64 true "идентификатор заказа"
// @Param body body order.UpdateStatusInput true "новый статус"
// @Success 200 {object} order.Order
// @Failure 400 {object} errorResponse
// @Failure 404 {object} errorResponse
// @Failure 409 {object} errorResponse
// @Router /api/v1/orders/{id}/status [patch]
func (h *Handler) updateOrderStatus(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{
			Error: "invalid order id",
			Code:  "INVALID_ID",
		})
		return
	}
	var input order.UpdateStatusInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{
			Error: "invalid json",
			Code:  "INVALID_REQUEST",
		})
		return
	}
	updateOrder, err := h.service.UpdateStatus(r.Context(), id, input.Status)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, updateOrder)
}

// @Summary получить историю статусов
// @Tags orders
// @Produce json
// @Param id path int64 true "идентификатор заказа"
// @Success 200 {array} order.StatusChange
// @Failure 400 {object} errorResponse
// @Failure 404 {object} errorResponse
// @Router /api/v1/orders/{id}/history [get]
func (h *Handler) getOrderHistory(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{
			Error: "invalid order id",
			Code:  "INVALID_ID",
		})
		return
	}
	history, err := h.service.History(r.Context(), id)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, history)
}

type errorResponse struct {
	Error string `json:"error"`
	Code  string `json:"code"`
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func (h *Handler) writeInternalError(w http.ResponseWriter, err error) {
	h.logger.Error("internal server error", slog.String("error", err.Error()))
	writeJSON(w, http.StatusInternalServerError, errorResponse{
		Error: "internal server error",
		Code:  "INTERNAL_ERROR",
	})
}

func parseID(r *http.Request) (int64, error) {
	return strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
}

func (h *Handler) writeServiceError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, order.ErrOrderNotFound):
		writeJSON(w, http.StatusNotFound, errorResponse{
			Error: "order not found",
			Code:  "ORDER_NOT_FOUND",
		})
	case errors.Is(err, order.ErrInvalidOrder),
		errors.Is(err, order.ErrInvalidStatus):
		writeJSON(w, http.StatusBadRequest, errorResponse{
			Error: "invalid request",
			Code:  "INVALID_REQUEST",
		})
	case errors.Is(err, order.ErrInvalidStatusTransition):
		writeJSON(w, http.StatusConflict, errorResponse{
			Error: "status transition is not allowed",
			Code:  "INVALID_STATUS_TRANSITION",
		})
	default:
		h.writeInternalError(w, err)
	}
}
