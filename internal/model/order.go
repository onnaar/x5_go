package model

const (
	StatusNew       = "new"
	StatusCancelled = "cancelled"
)

// Order сохраняет модель из предыдущего занятия.
type Order struct {
	ID       int
	Customer string
	Status   string
}
