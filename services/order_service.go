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
	couponSvc *CouponService
}

func NewOrderService(orderRepo repository.OrderRepository, couponSvc *CouponService) *OrderService {
	return &OrderService{
		orderRepo: orderRepo,
		couponSvc: couponSvc,
	}
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

	var subtotal float64
	for _, item := range order.Items {
		subtotal += item.Price * float64(item.Quantity)
	}

	order.SubtotalPrice = subtotal
	var discountAmount float64

	if order.CouponCode != "" && s.couponSvc != nil {
		_, disc, _, err := s.couponSvc.ValidateCoupon(ctx, order.CouponCode, subtotal)
		if err != nil {
			return nil, err
		}
		discountAmount = disc
		order.DiscountAmount = discountAmount
	}

	discountedSubtotal := subtotal - discountAmount
	if discountedSubtotal < 0 {
		discountedSubtotal = 0
	}

	// 5% GST tax calculation on discounted amount
	tax := discountedSubtotal * 0.05
	order.TotalPrice = float64(int64((discountedSubtotal+tax)*100+0.5)) / 100

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

	createdOrder, err := s.orderRepo.CreateOrder(ctx, order)
	if err != nil {
		return nil, err
	}

	// Mark single-use coupon as used
	if order.CouponCode != "" && s.couponSvc != nil {
		_ = s.couponSvc.MarkCouponUsed(ctx, order.CouponCode, createdOrder.ID)
	}

	return createdOrder, nil
}

func (s *OrderService) GetClientOrders(ctx context.Context, userID int64) ([]models.Order, error) {
	return s.orderRepo.GetOrdersByUserID(ctx, userID)
}

func (s *OrderService) GetAllOrders(ctx context.Context) ([]models.Order, error) {
	return s.orderRepo.GetAllOrders(ctx)
}

func (s *OrderService) UpdateOrderStatus(ctx context.Context, id int64, status string) error {
	return s.UpdateOrderStatusWithStaff(ctx, id, status, "", nil)
}

func (s *OrderService) UpdateOrderStatusWithStaff(ctx context.Context, id int64, status string, cancellationReason string, staff *models.User) error {
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

	if status == "Cancelled" && cancellationReason == "" {
		return errors.New("a reason is required when cancelling an order")
	}

	order, err := s.orderRepo.GetOrderByID(ctx, id)
	if err != nil {
		return err
	}

	// Superadmin / Admin restriction:
	// Superadmin/Admin must not claim an order directly. They must assign it to a staff member.
	if staff != nil && (staff.Role == "superadmin" || staff.Role == "admin") {
		if order.AssignedStaffID == 0 && status != "Cancelled" {
			return errors.New("As Superadmin/Admin, you cannot claim or fulfill orders directly. Please assign this order to a staff member to fulfill it.")
		}
	}

	// Duplicate protection logic:
	// If order is already claimed by another staff member, block staff update unless actor is admin/superadmin
	if staff != nil && order.AssignedStaffID != 0 && order.AssignedStaffID != staff.ID {
		if staff.Role != "superadmin" && staff.Role != "admin" {
			return fmt.Errorf("Order #%d is already claimed and being handled by %s", id, order.AssignedStaffName)
		}
	}

	staffID := order.AssignedStaffID
	staffName := order.AssignedStaffName

	// Staff members (role == "staff") can claim unassigned orders directly
	if staff != nil && staff.Role == "staff" {
		if staffID == 0 || staffID == staff.ID {
			staffID = staff.ID
			staffName = staff.Name
		}
	}

	return s.orderRepo.UpdateOrderStatusWithStaff(ctx, id, status, staffID, staffName, cancellationReason)
}

func (s *OrderService) AssignOrderToStaff(ctx context.Context, orderID int64, staffUser *models.User, assignedBy string) error {
	if staffUser == nil {
		return errors.New("invalid staff user")
	}

	if staffUser.Role != "staff" {
		return errors.New("orders can only be assigned to staff members")
	}

	order, err := s.orderRepo.GetOrderByID(ctx, orderID)
	if err != nil {
		return err
	}

	if order.Status == "Cancelled" || order.Status == "Completed" {
		return fmt.Errorf("cannot reassign an order that is already %s", order.Status)
	}

	return s.orderRepo.AssignOrderToStaff(ctx, orderID, staffUser.ID, staffUser.Name, assignedBy)
}

func (s *OrderService) GetCancellationReasons(ctx context.Context) ([]string, error) {
	return s.orderRepo.GetCancellationReasons(ctx)
}

func (s *OrderService) AddCancellationReason(ctx context.Context, reason string) error {
	return s.orderRepo.AddCancellationReason(ctx, reason)
}

func (s *OrderService) DeleteCancellationReason(ctx context.Context, reason string) error {
	return s.orderRepo.DeleteCancellationReason(ctx, reason)
}

func (s *OrderService) SubmitOrderReview(ctx context.Context, orderID int64, userID int64, rating int, review string) error {
	if rating < 1 || rating > 5 {
		return errors.New("rating must be between 1 and 5 stars")
	}

	order, err := s.orderRepo.GetOrderByID(ctx, orderID)
	if err != nil {
		return err
	}

	if userID != 0 && order.UserID != 0 && order.UserID != userID {
		return errors.New("unauthorized to review this order")
	}

	if order.Status != "Completed" {
		return errors.New("reviews can only be submitted for completed orders")
	}

	return s.orderRepo.SaveOrderReview(ctx, orderID, rating, review)
}
