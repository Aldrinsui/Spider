package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"github.com/Aldrinsui/Spider/internal/domain"
	"github.com/google/uuid"
)

type AuditRepository interface {
	Record(ctx context.Context, log *domain.AuditLog) error
	ListByTarget(ctx context.Context, targetType string, targetID uuid.UUID) ([]domain.AuditLog, error)
}

type sqlAuditRepository struct {
	db *sql.DB
}

func NewAuditRepository(db *sql.DB) AuditRepository {
	return &sqlAuditRepository{db: db}
}

func (r *sqlAuditRepository) Record(ctx context.Context, log *domain.AuditLog) error {
	query := `
		INSERT INTO audit_logs (id, admin_id, action, target_type, target_id, changes, ip_address, user_agent, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`
	if log.ID == uuid.Nil {
		log.ID = uuid.New()
	}
	log.CreatedAt = time.Now()

	_, err := r.db.ExecContext(ctx, query,
		log.ID,
		log.AdminID,
		log.Action,
		log.TargetType,
		log.TargetID,
		log.Changes,
		log.IPAddress,
		log.UserAgent,
		log.CreatedAt,
	)
	return err
}

func (r *sqlAuditRepository) ListByTarget(ctx context.Context, targetType string, targetID uuid.UUID) ([]domain.AuditLog, error) {
	query := `
		SELECT id, admin_id, action, target_type, target_id, changes, ip_address, user_agent, created_at
		FROM audit_logs
		WHERE target_type = $1 AND target_id = $2
		ORDER BY created_at DESC
	`
	rows, err := r.db.QueryContext(ctx, query, targetType, targetID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []domain.AuditLog
	for rows.Next() {
		var l domain.AuditLog
		var adminIDStr, ip, ua sql.NullString
		var changesBytes []byte

		if err := rows.Scan(
			&l.ID,
			&adminIDStr,
			&l.Action,
			&l.TargetType,
			&l.TargetID,
			&changesBytes,
			&ip,
			&ua,
			&l.CreatedAt,
		); err != nil {
			return nil, err
		}

		if adminIDStr.Valid {
			if id, err := uuid.Parse(adminIDStr.String); err == nil {
				l.AdminID = &id
			}
		}
		if ip.Valid {
			l.IPAddress = ip.String
		}
		if ua.Valid {
			l.UserAgent = ua.String
		}
		l.Changes = json.RawMessage(changesBytes)

		logs = append(logs, l)
	}

	return logs, nil
}
