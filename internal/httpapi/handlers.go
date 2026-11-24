package httpapi

import (
	"errors"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"

	"example.com/lesson9practice/internal/model"
	"example.com/lesson9practice/internal/repository"
	"example.com/lesson9practice/internal/service"
)

type Orders interface {
	Create(string) (model.Order, error)
	ByID(int) (model.Order, error)
	List(limit, offset int, status string) ([]model.Order, error)
}
type API struct {
	orders Orders
	logger *slog.Logger
}

func New(orders Orders, logger *slog.Logger) *API { return &API{orders: orders, logger: logger} }

var errInvalidQuery = errors.New("invalid query")

const (
	defaultLimit = 20
	minLimit     = 1
	maxLimit     = 100
)

// @Summary проверка статуса сервиса
// @Tags status
// @Produce json
// @Success 200 {object} StatusResponse
// @Router /status [get]
func (a *API) status(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, 200, StatusResponse{Status: "ok"})
}

// Дополнительно: задокументировать POST /orders.
func (a *API) createOrder(w http.ResponseWriter, r *http.Request) {
	// Указатель позволяет отличить null от объекта. Теги DTO сами не валидируют запрос.
	var in *CreateOrderRequest
	if err := decodeJSON(w, r, &in); err != nil {
		writeError(w, err.status, err.message, err.code)
		return
	}
	if in == nil {
		writeError(w, 400, "body must be a JSON object", "INVALID_JSON")
		return
	}
	order, err := a.orders.Create(in.Customer)
	if errors.Is(err, service.ErrEmptyCustomer) {
		writeError(w, 400, "customer is required", "VALIDATION_ERROR")
		return
	}
	if err != nil {
		a.internalError(w, r, err)
		return
	}
	w.Header().Set("Location", "/api/v1/orders/"+strconv.Itoa(order.ID))
	writeJSON(w, 201, orderResponse(order))
}

// @Summary список заказов
// @Description возвращает список заказов c пагинацией; сортировка всегда по id по возрастанию
// @Tags orders
// @Produce json
// @Param limit query int false "Размер страницы" minimum(1) maximum(100) default(20)
// @Param offset query int false "Смещение" minimum(0) default(0)
// @Success 200 {array} OrderResponse
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /orders [get]
func (a *API) listOrders(w http.ResponseWriter, r *http.Request) {
	limit, offset, err := parsePaginationQuery(r.URL.Query())
	if err != nil {
		writeError(w, 400, "invalid query parameters", "INVALID_QUERY")
		return
	}
	orders, err := a.orders.List(limit, offset, "")
	if err != nil {
		a.internalError(w, r, err)
		return
	}
	response := make([]OrderResponse, len(orders))
	for i := range orders {
		response[i] = orderResponse(orders[i])
	}
	writeJSON(w, 200, response)
}

// @Summary получение заказа по номеру
// @Description возвращает заказ по положительному числовому идентификатору
// @Tags orders
// @Produce json
// @Param id path int true "Номер заказа" minimum(1)
// @Success 200 {object} OrderResponse
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /orders/{id} [get]
func (a *API) getOrder(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil || id <= 0 {
		writeError(w, 400, "id must be a positive integer", "INVALID_ID")
		return
	}
	order, err := a.orders.ByID(id)
	if errors.Is(err, repository.ErrNotFound) {
		writeError(w, 404, "order not found", "ORDER_NOT_FOUND")
		return
	}
	if err != nil {
		a.internalError(w, r, err)
		return
	}
	writeJSON(w, 200, orderResponse(order))
}

func (a *API) internalError(w http.ResponseWriter, r *http.Request, err error) {
	a.logger.ErrorContext(r.Context(), "order operation failed", "error", err)
	writeError(w, 500, "internal server error", "INTERNAL_ERROR")
}

func parsePaginationQuery(query url.Values) (int, int, error) {
	for key, values := range query {
		if key != "limit" && key != "offset" {
			return 0, 0, errInvalidQuery
		}
		if len(values) != 1 {
			return 0, 0, errInvalidQuery
		}
	}
	limit := defaultLimit
	if value, ok := query["limit"]; ok {
		parsed, err := parseDecimalInt(value[0])
		if err != nil || parsed < minLimit || parsed > maxLimit {
			return 0, 0, errInvalidQuery
		}
		limit = parsed
	}
	offset := 0
	if value, ok := query["offset"]; ok {
		parsed, err := parseDecimalInt(value[0])
		if err != nil {
			return 0, 0, errInvalidQuery
		}
		offset = parsed
	}
	return limit, offset, nil
}

func parseDecimalInt(value string) (int, error) {
	if value == "" {
		return 0, errInvalidQuery
	}
	for i := 0; i < len(value); i++ {
		if value[i] < '0' || value[i] > '9' {
			return 0, errInvalidQuery
		}
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return 0, errInvalidQuery
	}
	return parsed, nil
}
