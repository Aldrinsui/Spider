package test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
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

type MockBookingRepository struct {
	mu            sync.Mutex
	bookings      map[uuid.UUID]*domain.Booking
	containerRepo *MockContainerRepository
	outboxRepo    *MockOutboxRepository
}

func NewMockBookingRepository(containerRepo *MockContainerRepository, outboxRepo *MockOutboxRepository) repository.BookingRepository {
	return &MockBookingRepository{
		bookings:      make(map[uuid.UUID]*domain.Booking),
		containerRepo: containerRepo,
		outboxRepo:    outboxRepo,
	}
}

func (m *MockBookingRepository) HoldSpaceTransaction(ctx context.Context, b *domain.Booking, outboxEvent *domain.OutboxEvent) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	container, ok := m.containerRepo.containers[b.ContainerID]
	if !ok {
		return errors.New("container not found")
	}

	if container.AvailableCBM < b.ChargeableVolumeCBM {
		return fmt.Errorf("insufficient capacity: requested %.2f CBM, available %.2f CBM", b.ChargeableVolumeCBM, container.AvailableCBM)
	}
	if container.AvailableWeightKG < b.ActualWeightKG {
		return fmt.Errorf("insufficient payload weight: requested %.2f kg, available %.2f kg", b.ActualWeightKG, container.AvailableWeightKG)
	}

	container.AvailableCBM -= b.ChargeableVolumeCBM
	container.AvailableWeightKG -= b.ActualWeightKG

	if b.ID == uuid.Nil {
		b.ID = uuid.New()
	}
	now := time.Now()
	exp := now.Add(10 * time.Minute)
	b.Status = domain.BookingStatusHeld
	b.HoldExpiresAt = &exp
	b.CreatedAt = now
	b.UpdatedAt = now
	m.bookings[b.ID] = b

	if outboxEvent != nil && m.outboxRepo != nil {
		if outboxEvent.ID == uuid.Nil {
			outboxEvent.ID = uuid.New()
		}
		outboxEvent.Status = domain.OutboxStatusPending
		outboxEvent.CreatedAt = now
		m.outboxRepo.events = append(m.outboxRepo.events, *outboxEvent)
	}

	return nil
}

func (m *MockBookingRepository) ReclaimExpiredHolds(ctx context.Context) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	reclaimed := 0
	now := time.Now()

	for _, b := range m.bookings {
		if b.Status == domain.BookingStatusHeld && b.HoldExpiresAt != nil && b.HoldExpiresAt.Before(now) {
			b.Status = domain.BookingStatusExpired
			if container, ok := m.containerRepo.containers[b.ContainerID]; ok {
				container.AvailableCBM += b.ChargeableVolumeCBM
				container.AvailableWeightKG += b.ActualWeightKG
			}
			reclaimed++
		}
	}
	return reclaimed, nil
}

func (m *MockBookingRepository) GetBookingByID(ctx context.Context, id uuid.UUID) (*domain.Booking, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if b, ok := m.bookings[id]; ok {
		return b, nil
	}
	return nil, nil
}

func (m *MockBookingRepository) GetBookingByRef(ctx context.Context, ref string) (*domain.Booking, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, b := range m.bookings {
		if b.BookingRef == ref {
			return b, nil
		}
	}
	return nil, nil
}

func (m *MockBookingRepository) ListBookingsByTrader(ctx context.Context, traderID uuid.UUID) ([]domain.Booking, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var res []domain.Booking
	for _, b := range m.bookings {
		if b.TraderID == traderID {
			res = append(res, *b)
		}
	}
	return res, nil
}

func (m *MockBookingRepository) ListBookingsByContainer(ctx context.Context, containerID uuid.UUID) ([]domain.Booking, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var res []domain.Booking
	for _, b := range m.bookings {
		if b.ContainerID == containerID {
			res = append(res, *b)
		}
	}
	return res, nil
}

func (m *MockBookingRepository) UpdateBookingStatus(ctx context.Context, id uuid.UUID, status domain.BookingStatus) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if b, ok := m.bookings[id]; ok {
		b.Status = status
		return nil
	}
	return errors.New("booking not found")
}

type MockOutboxRepository struct {
	mu     sync.Mutex
	events []domain.OutboxEvent
}

func NewMockOutboxRepository() *MockOutboxRepository {
	return &MockOutboxRepository{events: []domain.OutboxEvent{}}
}

func (m *MockOutboxRepository) GetPendingEvents(ctx context.Context, limit int) ([]domain.OutboxEvent, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var res []domain.OutboxEvent
	for _, e := range m.events {
		if e.Status == domain.OutboxStatusPending {
			res = append(res, e)
			if len(res) >= limit {
				break
			}
		}
	}
	return res, nil
}

func (m *MockOutboxRepository) MarkEventCompleted(ctx context.Context, id uuid.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for i := range m.events {
		if m.events[i].ID == id {
			m.events[i].Status = domain.OutboxStatusCompleted
			now := time.Now()
			m.events[i].ProcessedAt = &now
			return nil
		}
	}
	return errors.New("event not found")
}

func (m *MockOutboxRepository) MarkEventFailed(ctx context.Context, id uuid.UUID, lastErr string, retryCount int) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for i := range m.events {
		if m.events[i].ID == id {
			m.events[i].LastError = lastErr
			m.events[i].RetryCount = retryCount
			if retryCount >= 5 {
				m.events[i].Status = domain.OutboxStatusFailed
			}
			return nil
		}
	}
	return errors.New("event not found")
}

type MockEscrowRepository struct {
	mu            sync.Mutex
	escrows       map[uuid.UUID]*domain.PaymentEscrow
	processedPay  map[string]bool
	bookingRepo   *MockBookingRepository
	containerRepo *MockContainerRepository
	outboxRepo    *MockOutboxRepository
}

func NewMockEscrowRepository(bookingRepo *MockBookingRepository, containerRepo *MockContainerRepository, outboxRepo *MockOutboxRepository) repository.EscrowRepository {
	return &MockEscrowRepository{
		escrows:       make(map[uuid.UUID]*domain.PaymentEscrow),
		processedPay:  make(map[string]bool),
		bookingRepo:   bookingRepo,
		containerRepo: containerRepo,
		outboxRepo:    outboxRepo,
	}
}

func (m *MockEscrowRepository) CreatePaymentEscrow(ctx context.Context, e *domain.PaymentEscrow) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if e.ID == uuid.Nil {
		e.ID = uuid.New()
	}
	e.CreatedAt = time.Now()
	e.UpdatedAt = time.Now()
	m.escrows[e.BookingID] = e
	return nil
}

func (m *MockEscrowRepository) GetEscrowByBookingID(ctx context.Context, bookingID uuid.UUID) (*domain.PaymentEscrow, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if e, ok := m.escrows[bookingID]; ok {
		return e, nil
	}
	return nil, nil
}

func (m *MockEscrowRepository) GetEscrowByOrderID(ctx context.Context, orderID string) (*domain.PaymentEscrow, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, e := range m.escrows {
		if e.RazorpayOrderID == orderID {
			return e, nil
		}
	}
	return nil, nil
}

func (m *MockEscrowRepository) IsWebhookProcessed(ctx context.Context, paymentID string) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.processedPay[paymentID], nil
}

func (m *MockEscrowRepository) RecordProcessedWebhook(ctx context.Context, eventID, paymentID, eventType string, payload json.RawMessage) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.processedPay[paymentID] = true
	return nil
}

func (m *MockEscrowRepository) ConfirmPaymentAndHoldEscrow(ctx context.Context, bookingID uuid.UUID, paymentID, signature, routeTransferID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	b, ok := m.bookingRepo.bookings[bookingID]
	if !ok {
		return errors.New("booking not found")
	}
	b.Status = domain.BookingStatusConfirmed
	now := time.Now()
	b.ConfirmedAt = &now

	e, ok := m.escrows[bookingID]
	if !ok {
		return errors.New("escrow not found")
	}
	e.RazorpayPaymentID = paymentID
	e.RazorpaySignature = signature
	e.ProviderRouteTransferID = routeTransferID
	e.EscrowStatus = domain.EscrowStatusHeld

	if m.outboxRepo != nil {
		m.outboxRepo.events = append(m.outboxRepo.events, domain.OutboxEvent{
			ID:          uuid.New(),
			EventType:   "booking.confirmed",
			AggregateID: bookingID,
			Status:      domain.OutboxStatusPending,
			CreatedAt:   now,
		})
	}

	return nil
}

func (m *MockEscrowRepository) ReconcileLatePayment(ctx context.Context, bookingID uuid.UUID, paymentID, signature string) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	b, ok := m.bookingRepo.bookings[bookingID]
	if !ok {
		return false, errors.New("booking not found")
	}
	c, ok := m.containerRepo.containers[b.ContainerID]
	if !ok {
		return false, errors.New("container not found")
	}
	e, ok := m.escrows[bookingID]
	if !ok {
		return false, errors.New("escrow not found")
	}

	e.RazorpayPaymentID = paymentID
	e.RazorpaySignature = signature

	if c.AvailableCBM >= b.ChargeableVolumeCBM && c.AvailableWeightKG >= b.ActualWeightKG {
		c.AvailableCBM -= b.ChargeableVolumeCBM
		c.AvailableWeightKG -= b.ActualWeightKG
		b.Status = domain.BookingStatusConfirmed
		now := time.Now()
		b.ConfirmedAt = &now
		e.EscrowStatus = domain.EscrowStatusHeld
		return true, nil
	}

	b.Status = domain.BookingStatusRefunded
	now := time.Now()
	e.EscrowStatus = domain.EscrowStatusRefundedToTrader
	e.RefundedAt = &now
	return false, nil
}

func (m *MockEscrowRepository) ReleaseEscrowPayout(ctx context.Context, bookingID uuid.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	e, ok := m.escrows[bookingID]
	if !ok {
		return errors.New("escrow not found")
	}
	if e.EscrowStatus != domain.EscrowStatusHeld {
		return errors.New("escrow is not held")
	}
	now := time.Now()
	e.EscrowStatus = domain.EscrowStatusReleasedToProvider
	e.PayoutReleasedAt = &now
	return nil
}

func (m *MockEscrowRepository) ProcessRefund(ctx context.Context, bookingID uuid.UUID, refundID string, refundAmountPaise int64, reason string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	b, ok := m.bookingRepo.bookings[bookingID]
	if !ok {
		return errors.New("booking not found")
	}
	b.Status = domain.BookingStatusCancelled
	now := time.Now()
	b.CancelledAt = &now

	e, ok := m.escrows[bookingID]
	if !ok {
		return errors.New("escrow not found")
	}
	e.EscrowStatus = domain.EscrowStatusRefundedToTrader
	e.RefundID = refundID
	e.RefundAmountPaise = &refundAmountPaise
	e.RefundedAt = &now

	if c, ok := m.containerRepo.containers[b.ContainerID]; ok {
		c.AvailableCBM += b.ChargeableVolumeCBM
		c.AvailableWeightKG += b.ActualWeightKG
	}

	return nil
}

type MockChatRepository struct {
	mu       sync.Mutex
	messages map[uuid.UUID][]domain.ChatMessage
}

func NewMockChatRepository() repository.ChatRepository {
	return &MockChatRepository{
		messages: make(map[uuid.UUID][]domain.ChatMessage),
	}
}

func (m *MockChatRepository) SendMessage(ctx context.Context, msg *domain.ChatMessage) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if msg.ID == uuid.Nil {
		msg.ID = uuid.New()
	}
	msg.CreatedAt = time.Now()
	m.messages[msg.BookingID] = append(m.messages[msg.BookingID], *msg)
	return nil
}

func (m *MockChatRepository) ListMessages(ctx context.Context, bookingID uuid.UUID, limit int, before *time.Time) ([]domain.ChatMessage, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	all := m.messages[bookingID]
	var res []domain.ChatMessage
	for _, msg := range all {
		if before != nil && !msg.CreatedAt.Before(*before) {
			continue
		}
		res = append(res, msg)
		if len(res) >= limit {
			break
		}
	}
	return res, nil
}

func (m *MockChatRepository) MarkMessagesAsRead(ctx context.Context, bookingID uuid.UUID, readerID uuid.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for i := range m.messages[bookingID] {
		if m.messages[bookingID][i].SenderID != readerID {
			m.messages[bookingID][i].IsRead = true
		}
	}
	return nil
}

type MockDocumentRepository struct {
	mu   sync.Mutex
	docs map[uuid.UUID][]repository.DocumentVaultItem
}

func NewMockDocumentRepository() repository.DocumentRepository {
	return &MockDocumentRepository{
		docs: make(map[uuid.UUID][]repository.DocumentVaultItem),
	}
}

func (m *MockDocumentRepository) AddDocument(ctx context.Context, doc *repository.DocumentVaultItem) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if doc.ID == uuid.Nil {
		doc.ID = uuid.New()
	}
	doc.CreatedAt = time.Now()
	m.docs[doc.BookingID] = append(m.docs[doc.BookingID], *doc)
	return nil
}

func (m *MockDocumentRepository) ListDocumentsByBooking(ctx context.Context, bookingID uuid.UUID) ([]repository.DocumentVaultItem, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.docs[bookingID], nil
}

type MockTrackingRepository struct {
	mu            sync.Mutex
	proofs        map[uuid.UUID]*repository.DeliveryProof
	bookingRepo   *MockBookingRepository
	escrowRepo    *MockEscrowRepository
	containerRepo *MockContainerRepository
}

func NewMockTrackingRepository(bookingRepo *MockBookingRepository, escrowRepo *MockEscrowRepository, containerRepo *MockContainerRepository) repository.TrackingRepository {
	return &MockTrackingRepository{
		proofs:        make(map[uuid.UUID]*repository.DeliveryProof),
		bookingRepo:   bookingRepo,
		escrowRepo:    escrowRepo,
		containerRepo: containerRepo,
	}
}

func (m *MockTrackingRepository) UpdateContainerLocation(ctx context.Context, containerID uuid.UUID, lat, lng float64) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if c, ok := m.containerRepo.containers[containerID]; ok {
		c.CurrentLocation = &domain.LocationPoint{Latitude: lat, Longitude: lng}
		now := time.Now()
		c.LastLocationUpdatedAt = &now
		return nil
	}
	return errors.New("container not found")
}

func (m *MockTrackingRepository) SubmitDeliveryProof(ctx context.Context, proof *repository.DeliveryProof) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if proof.ID == uuid.Nil {
		proof.ID = uuid.New()
	}
	proof.DeliveredAt = time.Now()
	m.proofs[proof.BookingID] = proof

	if b, ok := m.bookingRepo.bookings[proof.BookingID]; ok {
		b.Status = domain.BookingStatusDelivered
		now := time.Now()
		b.DeliveredAt = &now
	}
	if e, ok := m.escrowRepo.escrows[proof.BookingID]; ok {
		e.EscrowStatus = domain.EscrowStatusReleasedToProvider
		now := time.Now()
		e.PayoutReleasedAt = &now
	}
	return nil
}

func (m *MockTrackingRepository) GetDeliveryProof(ctx context.Context, bookingID uuid.UUID) (*repository.DeliveryProof, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if p, ok := m.proofs[bookingID]; ok {
		return p, nil
	}
	return nil, nil
}

type MockDisputeRepository struct {
	mu          sync.Mutex
	disputes    map[uuid.UUID]*repository.DisputeItem
	bookingRepo *MockBookingRepository
	escrowRepo  *MockEscrowRepository
}

func NewMockDisputeRepository(bookingRepo *MockBookingRepository, escrowRepo *MockEscrowRepository) repository.DisputeRepository {
	return &MockDisputeRepository{
		disputes:    make(map[uuid.UUID]*repository.DisputeItem),
		bookingRepo: bookingRepo,
		escrowRepo:  escrowRepo,
	}
}

func (m *MockDisputeRepository) CreateDispute(ctx context.Context, d *repository.DisputeItem) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if d.ID == uuid.Nil {
		d.ID = uuid.New()
	}
	now := time.Now()
	d.CreatedAt = now
	d.UpdatedAt = now
	m.disputes[d.ID] = d

	if b, ok := m.bookingRepo.bookings[d.BookingID]; ok {
		b.Status = domain.BookingStatusDisputed
	}
	if e, ok := m.escrowRepo.escrows[d.BookingID]; ok {
		e.EscrowStatus = domain.EscrowStatusDisputedFreeze
	}
	return nil
}

func (m *MockDisputeRepository) GetDisputeByID(ctx context.Context, id uuid.UUID) (*repository.DisputeItem, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if d, ok := m.disputes[id]; ok {
		return d, nil
	}
	return nil, nil
}

func (m *MockDisputeRepository) ListDisputes(ctx context.Context, status string) ([]repository.DisputeItem, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var res []repository.DisputeItem
	for _, d := range m.disputes {
		if status == "" || d.Status == status {
			res = append(res, *d)
		}
	}
	return res, nil
}

func (m *MockDisputeRepository) ResolveDispute(ctx context.Context, id uuid.UUID, status string, notes string, adminID uuid.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	d, ok := m.disputes[id]
	if !ok {
		return errors.New("dispute not found")
	}
	d.Status = status
	d.ResolutionNotes = notes
	d.ResolvedBy = &adminID
	now := time.Now()
	d.ResolvedAt = &now

	if status == "resolved_refund" {
		if b, ok := m.bookingRepo.bookings[d.BookingID]; ok {
			b.Status = domain.BookingStatusRefunded
		}
		if e, ok := m.escrowRepo.escrows[d.BookingID]; ok {
			e.EscrowStatus = domain.EscrowStatusRefundedToTrader
		}
	} else if status == "resolved_payout" {
		if b, ok := m.bookingRepo.bookings[d.BookingID]; ok {
			b.Status = domain.BookingStatusDelivered
		}
		if e, ok := m.escrowRepo.escrows[d.BookingID]; ok {
			e.EscrowStatus = domain.EscrowStatusReleasedToProvider
		}
	}
	return nil
}
