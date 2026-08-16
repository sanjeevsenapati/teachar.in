package services

import (
	"context"
	"errors"
	"fmt"
	"time"

	"teachar.in/models"
	"teachar.in/repository"
)

type OrderService struct {
	orderRepo repository.OrderRepository
}

func NewOrderService(orderRepo repository.OrderRepository) *OrderService {
	return &OrderService{orderRepo: orderRepo}
}

func (s *OrderService) CreateOrder(ctx context.Context, order models.Order) (*models.Order, error) {
	if len(order.Items) == 0 {
		return nil, errors.New("cannot create an empty order")
	}

	if order.OrderType == "" {
		order.OrderType = "Dine-in"
	}

	switch order.OrderType {
	case "Dine-in":
		if order.TableNumber == "" {
			return nil, errors.New("table number is required for Dine-in orders")
		}
	case "Takeaway":
		if order.CustomerPhone == "" {
			return nil, errors.New("mobile number is required for Takeaway orders")
		}
	case "Delivery":
		if order.CustomerPhone == "" {
			return nil, errors.New("mobile number is required for Delivery orders")
		}
		if order.DeliveryAddress == "" {
			return nil, errors.New("delivery address is required for Delivery orders")
		}
	default:
		return nil, errors.New("invalid order type")
	}

	var calculatedTotal float64
	for _, item := range order.Items {
		calculatedTotal += item.Price * float64(item.Quantity)
	}

	// 5% GST tax calculation
	tax := calculatedTotal * 0.05
	order.TotalPrice = calculatedTotal + tax

	if order.PaymentMethod == "" {
		order.PaymentMethod = "UPI"
	}

	if order.TransactionID == "" {
		order.TransactionID = fmt.Sprintf("TXN%d", time.Now().UnixNano()/1e5)
	}

	if order.PaymentMethod == "COD" {
		order.PaymentStatus = "Pending"
	} else {
		order.PaymentStatus = "Paid"
	}

	return s.orderRepo.CreateOrder(ctx, order)
}

func (s *OrderService) GetClientOrders(ctx context.Context, userID int64) ([]models.Order, error) {
	return s.orderRepo.GetOrdersByUserID(ctx, userID)
}

func (s *OrderService) GetAllOrders(ctx context.Context) ([]models.Order, error) {
	return s.orderRepo.GetAllOrders(ctx)
}

func (s *OrderService) UpdateOrderStatus(ctx context.Context, id int64, status string) error {
	validStatuses := map[string]bool{
		"Pending":   true,
		"Preparing": true,
		"Ready":     true,
		"Completed": true,
		"Cancelled": true,
	}

	if !validStatuses[status] {
		return errors.New("invalid status value")
	}

	return s.orderRepo.UpdateOrderStatus(ctx, id, status)
}
