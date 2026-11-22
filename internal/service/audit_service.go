package service

import (
	"context"
	"encoding/json"

	"github.com/Aldrinsui/Spider/internal/domain"
	"github.com/Aldrinsui/Spider/internal/repository"
	"github.com/google/uuid"
)

type AuditService interface {
	Log(ctx context.Context, adminID *uuid.UUID, action, targetType string, targetID uuid.UUID, changes interface{}, ip, ua string) error
	GetTargetLogs(ctx context.Context, targetType string, targetID uuid.UUID) ([]domain.AuditLog, error)
}

type auditService struct {
	repo repository.AuditRepository
}

func NewAuditService(repo repository.AuditRepository) AuditService {
	return &auditService{repo: repo}
}

func (s *auditService) Log(ctx context.Context, adminID *uuid.UUID, action, targetType string, targetID uuid.UUID, changes interface{}, ip, ua string) error {
	raw, err := json.Marshal(changes)
	if err != nil {
		raw = []byte("{}")
	}

	entry := &domain.AuditLog{
		AdminID:    adminID,
		Action:     action,
		TargetType: targetType,
		TargetID:   targetID,
		Changes:    raw,
		IPAddress:  ip,
		UserAgent:  ua,
	}

	return s.repo.Record(ctx, entry)
}

func (s *auditService) GetTargetLogs(ctx context.Context, targetType string, targetID uuid.UUID) ([]domain.AuditLog, error) {
	return s.repo.ListByTarget(ctx, targetType, targetID)
}
