package services

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
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
	if name == "" || email == "" || mobileNumber == "" || password == "" {
		return nil, errors.New("all fields including mobile number are required")
	}

	salt := repository.GenerateSalt()
	passwordHash := repository.HashPassword(password, salt)

	user := models.User{
		Name:         name,
		Email:        email,
		MobileNumber: mobileNumber,
		PasswordHash: passwordHash,
		Salt:         salt,
		Role:         "client",
	}

	return s.userRepo.CreateUser(ctx, user)
}

func (s *AuthService) AuthenticateUser(ctx context.Context, email, password string) (*models.User, error) {
	user, err := s.userRepo.GetUserByEmail(ctx, email)
	if err != nil {
		return nil, errors.New("invalid email or password")
	}

	expectedHash := repository.HashPassword(password, user.Salt)
	if expectedHash != user.PasswordHash {
		return nil, errors.New("invalid email or password")
	}

	return user, nil
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

	return session, user, nil
}

func (s *AuthService) Logout(ctx context.Context, token string) error {
	return s.userRepo.DeleteSession(ctx, token)
}
