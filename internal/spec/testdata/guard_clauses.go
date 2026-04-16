package testdata

import (
	"errors"
	"fmt"
)

// Sentinel errors
var (
	ErrNotFound      = errors.New("resource not found")
	ErrUnauthorized  = errors.New("unauthorized access")
	ErrRateLimited   = errors.New("rate limit exceeded")
	ErrInvalidInput  = errors.New("invalid input")
)

func ProcessOrder(order Order) error {
	if order.Quantity <= 0 {
		return fmt.Errorf("quantity must be positive, got %d", order.Quantity)
	}
	if order.Amount < 0 {
		return fmt.Errorf("amount cannot be negative")
	}
	if len(order.Items) > 100 {
		return fmt.Errorf("too many items: max 100")
	}
	if order.CustomerID == "" {
		return errors.New("customer ID is required")
	}
	return nil
}

func ValidateUser(user User) error {
	if user.Age < 18 {
		return fmt.Errorf("user must be at least 18 years old")
	}
	if user.Email == "" {
		return errors.New("email is required")
	}
	return nil
}

type Order struct {
	Quantity   int
	Amount     float64
	Items      []string
	CustomerID string
}

type User struct {
	Age   int
	Email string
}
