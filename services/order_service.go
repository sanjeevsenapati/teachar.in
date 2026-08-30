package services

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"teachar.in/models"
	"teachar.in/repository"
)

type OrderService struct {
	orderRepo     repository.OrderRepository
	couponSvc     *CouponService
	membershipSvc *MembershipService
}

func NewOrderService(orderRepo repository.OrderRepository, couponSvc *CouponService, membershipSvc *MembershipService) *OrderService {
	return &OrderService{
		orderRepo:     orderRepo,
		couponSvc:     couponSvc,
		membershipSvc: membershipSvc,
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
			order.TableNumber = "Table 1"
		}
	case "Takeaway":
		if order.CustomerPhone == "" {
			order.CustomerPhone = "9876543210"
		}
	case "Delivery":
		if order.CustomerPhone == "" {
			order.CustomerPhone = "9876543210"
		}
		if order.DeliveryAddress == "" {
			return nil, errors.New("delivery address is required for Delivery orders")
		}
	default:
		order.OrderType = "Dine-in"
		order.TableNumber = "Table 1"
	}

	var subtotal float64
	for _, item := range order.Items {
		subtotal += item.Price * float64(item.Quantity)
	}

	order.SubtotalPrice = subtotal
	var couponDiscount float64
	var subscriberDiscount float64

	// 1. Member Subscription Price Adjustment & Automated VIP Coupon Attachment
	if order.UserID > 0 && s.membershipSvc != nil {
		sub, err := s.membershipSvc.GetUserSubscription(ctx, order.UserID)
		if err == nil && sub != nil && sub.Status == "Active" {
			subscriberDiscount = (subtotal * sub.DiscountPercent) / 100.0
			order.SubscriberDiscount = subscriberDiscount
			order.SubscriberTierName = sub.TierName

			// Automatically assign member VIP coupon code if not manually set
			if order.CouponCode == "" {
				switch sub.TierID {
				case "silver":
					order.CouponCode = "SILVERVIP"
				case "gold":
					order.CouponCode = "GOLDVIP"
				case "platinum":
					order.CouponCode = "PLATINUMVIP"
				}
			}

			// Auto claim daily free beverage credit if eligible
			_ = s.membershipSvc.ClaimDailyCup(ctx, order.UserID)
		}
	}

	// 2. Custom Coupon Discount Check (if not virtual VIP pass coupon)
	if order.CouponCode != "" && s.couponSvc != nil {
		if !strings.HasSuffix(order.CouponCode, "VIP") && !strings.HasSuffix(order.CouponCode, "PASS") && !strings.HasPrefix(order.CouponCode, "VIP") {
			_, disc, _, err := s.couponSvc.ValidateCouponForUser(ctx, order.CouponCode, subtotal, order.UserID)
			if err == nil {
				couponDiscount = disc
			} else {
				return nil, err
			}
		}
	}

	totalDiscount := couponDiscount + subscriberDiscount
	order.DiscountAmount = totalDiscount

	discountedSubtotal := subtotal - totalDiscount
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

	// Duplicate protection logic:
	// If order is already claimed by another staff member, block non-admin staff update
	if staff != nil && order.AssignedStaffID != 0 && order.AssignedStaffID != staff.ID {
		if staff.Role != "superadmin" && staff.Role != "admin" {
			return fmt.Errorf("Order #%d is already claimed and being handled by %s", id, order.AssignedStaffName)
		}
	}

	staffID := order.AssignedStaffID
	staffName := order.AssignedStaffName

	// Auto-assign staff or admin if order is currently unassigned
	if staff != nil {
		if staffID == 0 || staffID == staff.ID {
			staffID = staff.ID
			staffName = staff.Name
		}
	}

	return s.orderRepo.UpdateOrderStatusWithStaff(ctx, id, status, staffID, staffName, cancellationReason)
}

func (s *OrderService) AssignOrderToStaff(ctx context.Context, orderID int64, staffUser *models.User, assignedBy string, estimatedMinutes int) error {
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

	if estimatedMinutes <= 0 {
		estimatedMinutes = 20
	}

	return s.orderRepo.AssignOrderToStaff(ctx, orderID, staffUser.ID, staffUser.Name, assignedBy, estimatedMinutes)
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
