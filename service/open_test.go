package service_test

import (
	"errors"
	"testing"

	"example.com/delivery/model"
	"example.com/delivery/repository"
	"example.com/delivery/service"
)

func TestCalculatorCourier(t *testing.T) {
	calculator := service.NewCalculator(repository.Default())
	order := model.Order{ID: "order-1", Tariff: "courier", WeightGrams: 2_500, DistanceKM: 10}

	got, err := calculator.Calculate(order)
	if err != nil {
		t.Fatalf("Calculate() error = %v", err)
	}
	const want int64 = 50_000
	if got != want {
		t.Fatalf("Calculate() = %d, want %d", got, want)
	}
}

func TestCalculatorUnknownTariff(t *testing.T) {
	calculator := service.NewCalculator(repository.Default())
	_, err := calculator.Calculate(model.Order{ID: "order-2", Tariff: "drone", WeightGrams: 1_000, DistanceKM: 5})

	if !errors.Is(err, service.ErrUnknownTariff) {
		t.Fatalf("errors.Is(%v, ErrUnknownTariff) = false", err)
	}
	var priceErr *service.OrderPriceError
	if !errors.As(err, &priceErr) {
		t.Fatalf("errors.As(%v, *OrderPriceError) = false", err)
	}
}

func TestCalculatorInvalidWeight(t *testing.T) {
	calculator := service.NewCalculator(repository.Default())
	_, err := calculator.Calculate(model.Order{
		ID:          "order-3",
		Tariff:      "courier",
		WeightGrams: 0,
		DistanceKM:  10,
	})

	if !errors.Is(err, service.ErrInvalidWeight) {
		t.Fatalf("errors.Is(%v, ErrInvalidWeight) = false", err)
	}
	var priceErr *service.OrderPriceError
	if !errors.As(err, &priceErr) {
		t.Fatalf("errors.As(%v, *OrderPriceError) = false", err)
	}
}
