package test

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/Aldrinsui/Spider/internal/domain"
	"github.com/Aldrinsui/Spider/internal/repository"
	"github.com/google/uuid"
)

type MockUserRepository struct {
	mu    sync.Mutex
	users map[uuid.UUID]*domain.User
	otps  map[uuid.UUID]*domain.OTPVerification
}

func NewMockUserRepository() repository.UserRepository {
	return &MockUserRepository{
		users: make(map[uuid.UUID]*domain.User),
		otps:  make(map[uuid.UUID]*domain.OTPVerification),
	}
}

func (m *MockUserRepository) CreateUser(ctx context.Context, user *domain.User) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if user.ID == uuid.Nil {
		user.ID = uuid.New()
	}
	user.CreatedAt = time.Now()
	user.UpdatedAt = time.Now()
	m.users[user.ID] = user
	return nil
}

func (m *MockUserRepository) GetUserByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if u, ok := m.users[id]; ok {
		return u, nil
	}
	return nil, nil
}

func (m *MockUserRepository) GetUserByPhone(ctx context.Context, phone string) (*domain.User, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, u := range m.users {
		if u.Phone == phone {
			return u, nil
		}
	}
	return nil, nil
}

func (m *MockUserRepository) UpdateUser(ctx context.Context, user *domain.User) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	user.UpdatedAt = time.Now()
	m.users[user.ID] = user
	return nil
}

func (m *MockUserRepository) RecordDPDPConsent(ctx context.Context, userID uuid.UUID, version string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if u, ok := m.users[userID]; ok {
		now := time.Now()
		u.DPDPConsentAt = &now
		u.DPDPConsentVersion = version
		return nil
	}
	return errors.New("user not found")
}

func (m *MockUserRepository) CreateOTP(ctx context.Context, otp *domain.OTPVerification) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if otp.ID == uuid.Nil {
		otp.ID = uuid.New()
	}
	otp.CreatedAt = time.Now()
	m.otps[otp.ID] = otp
	return nil
}

func (m *MockUserRepository) GetLatestActiveOTP(ctx context.Context, phone string) (*domain.OTPVerification, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var latest *domain.OTPVerification
	for _, o := range m.otps {
		if o.Phone == phone && o.VerifiedAt == nil && o.ExpiresAt.After(time.Now()) {
			if latest == nil || o.CreatedAt.After(latest.CreatedAt) {
				latest = o
			}
		}
	}
	return latest, nil
}

func (m *MockUserRepository) IncrementOTPAttempts(ctx context.Context, id uuid.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if o, ok := m.otps[id]; ok {
		o.Attempts++
		return nil
	}
	return errors.New("otp not found")
}

func (m *MockUserRepository) MarkOTPVerified(ctx context.Context, id uuid.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if o, ok := m.otps[id]; ok {
		now := time.Now()
		o.VerifiedAt = &now
		return nil
	}
	return errors.New("otp not found")
}

type MockKYCRepository struct {
	mu        sync.Mutex
	profiles  map[uuid.UUID]*domain.ProviderKYC
	documents map[uuid.UUID][]domain.ProviderDocument
}

func NewMockKYCRepository() repository.KYCRepository {
	return &MockKYCRepository{
		profiles:  make(map[uuid.UUID]*domain.ProviderKYC),
		documents: make(map[uuid.UUID][]domain.ProviderDocument),
	}
}

func (m *MockKYCRepository) CreateOrUpdateKYC(ctx context.Context, kyc *domain.ProviderKYC) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if kyc.ID == uuid.Nil {
		kyc.ID = uuid.New()
	}
	kyc.CreatedAt = time.Now()
	kyc.UpdatedAt = time.Now()
	m.profiles[kyc.ID] = kyc
	return nil
}

func (m *MockKYCRepository) GetKYCByUserID(ctx context.Context, userID uuid.UUID) (*domain.ProviderKYC, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, p := range m.profiles {
		if p.UserID == userID {
			p.Documents = m.documents[p.ID]
			return p, nil
		}
	}
	return nil, nil
}

func (m *MockKYCRepository) GetKYCByID(ctx context.Context, id uuid.UUID) (*domain.ProviderKYC, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if p, ok := m.profiles[id]; ok {
		p.Documents = m.documents[id]
		return p, nil
	}
	return nil, nil
}

func (m *MockKYCRepository) ListPendingKYC(ctx context.Context, limit, offset int) ([]domain.ProviderKYC, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var list []domain.ProviderKYC
	for _, p := range m.profiles {
		if p.Status == domain.KYCStatusPendingReview {
			p.Documents = m.documents[p.ID]
			list = append(list, *p)
		}
	}
	return list, nil
}

func (m *MockKYCRepository) UpdateKYCStatus(ctx context.Context, kycID uuid.UUID, status domain.KYCStatus, rejectionReason string, reviewedBy *uuid.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if p, ok := m.profiles[kycID]; ok {
		p.Status = status
		p.RejectionReason = rejectionReason
		p.ReviewedBy = reviewedBy
		now := time.Now()
		p.ReviewedAt = &now
		return nil
	}
	return errors.New("profile not found")
}

func (m *MockKYCRepository) AddDocument(ctx context.Context, doc *domain.ProviderDocument) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if doc.ID == uuid.Nil {
		doc.ID = uuid.New()
	}
	doc.CreatedAt = time.Now()
	m.documents[doc.ProviderKYCID] = append(m.documents[doc.ProviderKYCID], *doc)
	return nil
}

func (m *MockKYCRepository) GetDocumentsByKYCID(ctx context.Context, kycID uuid.UUID) ([]domain.ProviderDocument, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.documents[kycID], nil
}

type MockAuditRepository struct {
	mu   sync.Mutex
	logs []domain.AuditLog
}

func NewMockAuditRepository() repository.AuditRepository {
	return &MockAuditRepository{logs: []domain.AuditLog{}}
}

func (m *MockAuditRepository) Record(ctx context.Context, log *domain.AuditLog) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if log.ID == uuid.Nil {
		log.ID = uuid.New()
	}
	log.CreatedAt = time.Now()
	m.logs = append(m.logs, *log)
	return nil
}

func (m *MockAuditRepository) ListByTarget(ctx context.Context, targetType string, targetID uuid.UUID) ([]domain.AuditLog, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var res []domain.AuditLog
	for _, l := range m.logs {
		if l.TargetType == targetType && l.TargetID == targetID {
			res = append(res, l)
		}
	}
	return res, nil
}

type MockContainerRepository struct {
	mu         sync.Mutex
	containers map[uuid.UUID]*domain.Container
}

func NewMockContainerRepository() repository.ContainerRepository {
	return &MockContainerRepository{
		containers: make(map[uuid.UUID]*domain.Container),
	}
}

func (m *MockContainerRepository) CreateContainer(ctx context.Context, c *domain.Container) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if c.ID == uuid.Nil {
		c.ID = uuid.New()
	}
	c.CreatedAt = time.Now()
	c.UpdatedAt = time.Now()
	m.containers[c.ID] = c
	return nil
}

func (m *MockContainerRepository) GetContainerByID(ctx context.Context, id uuid.UUID) (*domain.Container, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if c, ok := m.containers[id]; ok {
		return c, nil
	}
	return nil, nil
}

func (m *MockContainerRepository) UpdateContainer(ctx context.Context, c *domain.Container) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	c.UpdatedAt = time.Now()
	m.containers[c.ID] = c
	return nil
}

func (m *MockContainerRepository) SearchContainers(ctx context.Context, filter repository.ContainerSearchFilter) ([]domain.Container, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var res []domain.Container
	for _, c := range m.containers {
		if filter.MinCBM > 0 && c.AvailableCBM < filter.MinCBM {
			continue
		}
		if filter.TransportMode != "" && string(c.TransportMode) != filter.TransportMode {
			continue
		}
		res = append(res, *c)
	}
	return res, nil
}

func (m *MockContainerRepository) ListByProvider(ctx context.Context, providerID uuid.UUID) ([]domain.Container, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var res []domain.Container
	for _, c := range m.containers {
		if c.ProviderID == providerID {
			res = append(res, *c)
		}
	}
	return res, nil
}
