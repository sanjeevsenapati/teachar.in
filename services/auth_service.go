package services

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"strings"
	"time"

	"teachar.in/models"
	"teachar.in/repository"
)

type AuthService struct {
	userRepo repository.UserRepository
}

func NewAuthService(userRepo repository.UserRepository) *AuthService {
	return &AuthService{userRepo: userRepo}
}

func (s *AuthService) RegisterUser(ctx context.Context, name, email, mobileNumber, password string) (*models.User, error) {
	return s.CreateUserWithRole(ctx, name, email, mobileNumber, password, "client")
}

func (s *AuthService) CreateUserWithRole(ctx context.Context, name, email, mobileNumber, password, role string) (*models.User, error) {
	if name == "" || email == "" || mobileNumber == "" || password == "" {
		return nil, errors.New("all fields including mobile number are required")
	}

	if role == "" {
		role = "client"
	}

	salt := repository.GenerateSalt()
	passwordHash := repository.HashPassword(password, salt)

	user := models.User{
		Name:         name,
		Email:        email,
		MobileNumber: mobileNumber,
		PasswordHash: passwordHash,
		Salt:         salt,
		Role:         role,
	}

	return s.userRepo.CreateUser(ctx, user)
}

func (s *AuthService) GetAllUsers(ctx context.Context) ([]models.User, error) {
	return s.userRepo.GetAllUsers(ctx)
}

func (s *AuthService) GetUserByID(ctx context.Context, id int64) (*models.User, error) {
	return s.userRepo.GetUserByID(ctx, id)
}

func (s *AuthService) UpdateUserProfile(ctx context.Context, userID int64, name, email, mobileNumber, address, avatar string) (*models.User, error) {
	user, err := s.userRepo.GetUserByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	if name != "" {
		user.Name = name
	}
	if email != "" {
		user.Email = email
	}
	if mobileNumber != "" {
		user.MobileNumber = mobileNumber
	}
	if address != "" {
		user.Address = address
	}
	if avatar != "" {
		user.Avatar = avatar
	}

	return s.userRepo.UpdateUser(ctx, *user)
}

func (s *AuthService) AuthenticateUser(ctx context.Context, email, password string) (*models.User, error) {
	user, err := s.userRepo.GetUserByEmail(ctx, email)
	if err != nil {
		return nil, errors.New("invalid email or password")
	}

	if user.IsLocked || user.Status == "Locked" {
		return nil, errors.New("this user account is locked. Please contact system administrator")
	}

	if user.Status == "Disabled" {
		return nil, errors.New("this user account has been permanently disabled")
	}

	expectedHash := repository.HashPassword(password, user.Salt)
	if expectedHash != user.PasswordHash {
		return nil, errors.New("invalid email or password")
	}

	return user, nil
}

// AdminChangeUserPassword changes/resets password for a staff or admin user.
func (s *AuthService) AdminChangeUserPassword(ctx context.Context, targetUserID int64, newPassword string, actor *models.User) error {
	if actor == nil || (actor.Role != "admin" && actor.Role != "superadmin") {
		return errors.New("unauthorized: admin privileges required")
	}

	if strings.TrimSpace(newPassword) == "" {
		return errors.New("new password cannot be empty")
	}

	target, err := s.userRepo.GetUserByID(ctx, targetUserID)
	if err != nil {
		return errors.New("target user account not found")
	}

	if target.Role == "superadmin" || (target.Role == "admin" && actor.Role != "superadmin") {
		if actor.ID != target.ID {
			return errors.New("insufficient privileges to modify this admin account")
		}
	}

	salt := repository.GenerateSalt()
	passwordHash := repository.HashPassword(newPassword, salt)

	target.Salt = salt
	target.PasswordHash = passwordHash
	_, err = s.userRepo.UpdateUser(ctx, *target)
	return err
}

// AdminToggleUserLock locks or unlocks a staff or user account.
func (s *AuthService) AdminToggleUserLock(ctx context.Context, targetUserID int64, lockState bool, actor *models.User) error {
	if actor == nil || (actor.Role != "admin" && actor.Role != "superadmin") {
		return errors.New("unauthorized: admin privileges required")
	}

	if actor.ID == targetUserID {
		return errors.New("you cannot lock your own account")
	}

	target, err := s.userRepo.GetUserByID(ctx, targetUserID)
	if err != nil {
		return errors.New("target user account not found")
	}

	if target.Role == "superadmin" || (target.Role == "admin" && actor.Role != "superadmin") {
		return errors.New("insufficient privileges to lock this admin account")
	}

	target.IsLocked = lockState
	if lockState {
		target.Status = "Locked"
	} else {
		target.Status = "Active"
	}

	_, err = s.userRepo.UpdateUser(ctx, *target)
	return err
}

// AdminSetUserStatus sets user account status ("Active", "Locked", "Disabled").
func (s *AuthService) AdminSetUserStatus(ctx context.Context, targetUserID int64, status string, actor *models.User) error {
	if actor == nil || (actor.Role != "admin" && actor.Role != "superadmin") {
		return errors.New("unauthorized: admin privileges required")
	}

	if actor.ID == targetUserID {
		return errors.New("you cannot change status of your own account")
	}

	target, err := s.userRepo.GetUserByID(ctx, targetUserID)
	if err != nil {
		return errors.New("target user account not found")
	}

	if target.Role == "superadmin" || (target.Role == "admin" && actor.Role != "superadmin") {
		return errors.New("insufficient privileges to modify this admin account")
	}

	if status != "Active" && status != "Locked" && status != "Disabled" {
		return errors.New("invalid status value")
	}

	target.Status = status
	target.IsLocked = (status == "Locked" || status == "Disabled")

	_, err = s.userRepo.UpdateUser(ctx, *target)
	return err
}

// AdminDeleteUser permanently deletes a staff or user account.
func (s *AuthService) AdminDeleteUser(ctx context.Context, targetUserID int64, actor *models.User) error {
	if actor == nil || (actor.Role != "admin" && actor.Role != "superadmin") {
		return errors.New("unauthorized: admin privileges required")
	}

	if actor.ID == targetUserID {
		return errors.New("you cannot delete your own account")
	}

	target, err := s.userRepo.GetUserByID(ctx, targetUserID)
	if err != nil {
		return errors.New("target user account not found")
	}

	if target.Role == "superadmin" || (target.Role == "admin" && actor.Role != "superadmin") {
		return errors.New("insufficient privileges to delete this admin account")
	}

	return s.userRepo.DeleteUser(ctx, targetUserID)
}

func (s *AuthService) CreateSession(ctx context.Context, userID int64) (*models.Session, error) {
	tokenBytes := make([]byte, 32)
	rand.Read(tokenBytes)
	token := hex.EncodeToString(tokenBytes)

	session := models.Session{
		ID:        token,
		UserID:    userID,
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}

	if err := s.userRepo.CreateSession(ctx, session); err != nil {
		return nil, err
	}
	return &session, nil
}

func (s *AuthService) ValidateSession(ctx context.Context, token string) (*models.Session, *models.User, error) {
	if token == "" {
		return nil, nil, errors.New("no session token")
	}

	session, err := s.userRepo.GetSession(ctx, token)
	if err != nil {
		return nil, nil, err
	}

	user, err := s.userRepo.GetUserByID(ctx, session.UserID)
	if err != nil {
		return nil, nil, err
	}

	if user.IsLocked || user.Status == "Locked" || user.Status == "Disabled" {
		return nil, nil, errors.New("account is locked or disabled")
	}

	return session, user, nil
}

func (s *AuthService) Logout(ctx context.Context, token string) error {
	return s.userRepo.DeleteSession(ctx, token)
}
