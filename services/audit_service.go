package services

import (
	"context"
	"net/http"
	"strings"

	"teachar.in/models"
	"teachar.in/repository"
)

type AuditService struct {
	auditRepo repository.AuditRepository
}

func NewAuditService(auditRepo repository.AuditRepository) *AuditService {
	return &AuditService{auditRepo: auditRepo}
}

func (s *AuditService) LogEvent(ctx context.Context, actor *models.User, action, details, ip string) (*models.AuditLog, error) {
	var actorID int64 = 0
	var actorName = "Anonymous / System"
	var actorRole = "guest"

	if actor != nil {
		actorID = actor.ID
		actorName = actor.Name
		actorRole = actor.Role
	}

	if ip == "" {
		ip = "127.0.0.1"
	}

	log := models.AuditLog{
		ActorID:   actorID,
		ActorName: actorName,
		ActorRole: actorRole,
		Action:    action,
		Details:   details,
		IPAddress: ip,
	}

	return s.auditRepo.CreateAuditLog(ctx, log)
}

func (s *AuditService) GetAllLogs(ctx context.Context) ([]models.AuditLog, error) {
	return s.auditRepo.GetAllAuditLogs(ctx)
}

// GetClientIP extracts real client IP from HTTP Request headers or RemoteAddr
func GetClientIP(r *http.Request) string {
	forwarded := r.Header.Get("X-Forwarded-For")
	if forwarded != "" {
		parts := strings.Split(forwarded, ",")
		return strings.TrimSpace(parts[0])
	}
	realIP := r.Header.Get("X-Real-IP")
	if realIP != "" {
		return realIP
	}
	return r.RemoteAddr
}
