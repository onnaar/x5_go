package service

import "example.com/delivery/model"

type TariffRepository interface {
	Find(name string) (DeliveryPricer, error)
}

type Calculator struct {
	repository TariffRepository
}

func NewCalculator(repository TariffRepository) *Calculator {
	return &Calculator{repository: repository}
}

func (c *Calculator) Calculate(order model.Order) (int64, error) {
	pricer, err := c.repository.Find(order.Tariff)
	if err != nil {
		return 0, &OrderPriceError{order.ID, order.Tariff, err}
	}
	price, err := pricer.Price(order.WeightGrams, order.Distance)
	if err != nil {
		return 0, &OrderPriceError{order.ID, order.Tariff, err}
	}
	return price, nil
}
