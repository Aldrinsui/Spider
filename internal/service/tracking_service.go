package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/Aldrinsui/Spider/internal/repository"
	"github.com/google/uuid"
)

type TrackingService interface {
	UpdateLocation(ctx context.Context, providerID uuid.UUID, containerID uuid.UUID, lat, lng float64) error
	SubmitProofOfDelivery(ctx context.Context, providerID uuid.UUID, bookingID uuid.UUID, podPhotoURL, recipientName, recipientPhone, sigURL string, lat, lng *float64) (*repository.DeliveryProof, error)
	GetDeliveryProof(ctx context.Context, bookingID uuid.UUID) (*repository.DeliveryProof, error)
	UploadDocument(ctx context.Context, userID uuid.UUID, bookingID uuid.UUID, docType, fileURL, fileName string, fileSize int64, mimeType string) (*repository.DocumentVaultItem, error)
	ListDocuments(ctx context.Context, userID uuid.UUID, bookingID uuid.UUID) ([]repository.DocumentVaultItem, error)
	RaiseDispute(ctx context.Context, userID uuid.UUID, bookingID uuid.UUID, reason, description string, evidenceURLs []string) (*repository.DisputeItem, error)
	ResolveDispute(ctx context.Context, adminID uuid.UUID, disputeID uuid.UUID, status, notes string) error
	ListDisputes(ctx context.Context, status string) ([]repository.DisputeItem, error)
}

type trackingService struct {
	trackingRepo  repository.TrackingRepository
	docRepo       repository.DocumentRepository
	disputeRepo   repository.DisputeRepository
	bookingRepo   repository.BookingRepository
	containerRepo repository.ContainerRepository
}

func NewTrackingService(
	trackingRepo repository.TrackingRepository,
	docRepo repository.DocumentRepository,
	disputeRepo repository.DisputeRepository,
	bookingRepo repository.BookingRepository,
	containerRepo repository.ContainerRepository,
) TrackingService {
	return &trackingService{
		trackingRepo:  trackingRepo,
		docRepo:       docRepo,
		disputeRepo:   disputeRepo,
		bookingRepo:   bookingRepo,
		containerRepo: containerRepo,
	}
}

func (s *trackingService) UpdateLocation(ctx context.Context, providerID uuid.UUID, containerID uuid.UUID, lat, lng float64) error {
	container, err := s.containerRepo.GetContainerByID(ctx, containerID)
	if err != nil || container == nil {
		return errors.New("container not found")
	}
	if container.ProviderID != providerID {
		return errors.New("unauthorized: you do not own this container")
	}

	return s.trackingRepo.UpdateContainerLocation(ctx, containerID, lat, lng)
}

func (s *trackingService) SubmitProofOfDelivery(ctx context.Context, providerID uuid.UUID, bookingID uuid.UUID, podPhotoURL, recipientName, recipientPhone, sigURL string, lat, lng *float64) (*repository.DeliveryProof, error) {
	if podPhotoURL == "" {
		return nil, errors.New("proof-of-delivery (POD) photo is strictly required")
	}
	if recipientName == "" || recipientPhone == "" {
		return nil, errors.New("recipient name and phone number are required")
	}

	booking, err := s.bookingRepo.GetBookingByID(ctx, bookingID)
	if err != nil || booking == nil {
		return nil, errors.New("booking not found")
	}

	container, err := s.containerRepo.GetContainerByID(ctx, booking.ContainerID)
	if err != nil || container == nil {
		return nil, errors.New("container not found")
	}

	if container.ProviderID != providerID {
		return nil, errors.New("unauthorized: only the assigned logistics provider can submit proof of delivery")
	}

	proof := &repository.DeliveryProof{
		BookingID:             bookingID,
		PODPhotoURL:           podPhotoURL,
		Latitude:              lat,
		Longitude:             lng,
		RecipientName:         recipientName,
		RecipientPhone:        recipientPhone,
		RecipientSignatureURL: sigURL,
		ConfirmedByTrader:     false,
	}

	if err := s.trackingRepo.SubmitDeliveryProof(ctx, proof); err != nil {
		return nil, fmt.Errorf("failed to save proof of delivery: %w", err)
	}

	return proof, nil
}

func (s *trackingService) GetDeliveryProof(ctx context.Context, bookingID uuid.UUID) (*repository.DeliveryProof, error) {
	return s.trackingRepo.GetDeliveryProof(ctx, bookingID)
}

func (s *trackingService) UploadDocument(ctx context.Context, userID uuid.UUID, bookingID uuid.UUID, docType, fileURL, fileName string, fileSize int64, mimeType string) (*repository.DocumentVaultItem, error) {
	booking, err := s.bookingRepo.GetBookingByID(ctx, bookingID)
	if err != nil || booking == nil {
		return nil, errors.New("booking not found")
	}

	container, err := s.containerRepo.GetContainerByID(ctx, booking.ContainerID)
	if err != nil || container == nil {
		return nil, errors.New("container not found")
	}

	if booking.TraderID != userID && container.ProviderID != userID {
		return nil, errors.New("unauthorized: not a participant in this booking")
	}

	doc := &repository.DocumentVaultItem{
		BookingID:     bookingID,
		DocType:       docType,
		FileURL:       fileURL,
		FileName:      fileName,
		FileSizeBytes: fileSize,
		MimeType:      mimeType,
		UploadedBy:    userID,
	}

	if err := s.docRepo.AddDocument(ctx, doc); err != nil {
		return nil, err
	}

	return doc, nil
}

func (s *trackingService) ListDocuments(ctx context.Context, userID uuid.UUID, bookingID uuid.UUID) ([]repository.DocumentVaultItem, error) {
	booking, err := s.bookingRepo.GetBookingByID(ctx, bookingID)
	if err != nil || booking == nil {
		return nil, errors.New("booking not found")
	}

	container, err := s.containerRepo.GetContainerByID(ctx, booking.ContainerID)
	if err != nil || container == nil {
		return nil, errors.New("container not found")
	}

	if booking.TraderID != userID && container.ProviderID != userID {
		return nil, errors.New("unauthorized")
	}

	return s.docRepo.ListDocumentsByBooking(ctx, bookingID)
}

func (s *trackingService) RaiseDispute(ctx context.Context, userID uuid.UUID, bookingID uuid.UUID, reason, description string, evidenceURLs []string) (*repository.DisputeItem, error) {
	booking, err := s.bookingRepo.GetBookingByID(ctx, bookingID)
	if err != nil || booking == nil {
		return nil, errors.New("booking not found")
	}

	container, err := s.containerRepo.GetContainerByID(ctx, booking.ContainerID)
	if err != nil || container == nil {
		return nil, errors.New("container not found")
	}

	if booking.TraderID != userID && container.ProviderID != userID {
		return nil, errors.New("unauthorized: you must be a booking participant to raise a dispute")
	}

	rawEv, err := json.Marshal(evidenceURLs)
	if err != nil {
		rawEv = []byte("[]")
	}

	item := &repository.DisputeItem{
		BookingID:    bookingID,
		RaisedBy:     userID,
		Reason:       reason,
		Description:  description,
		EvidenceURLs: rawEv,
		Status:       "open",
	}

	if err := s.disputeRepo.CreateDispute(ctx, item); err != nil {
		return nil, fmt.Errorf("failed to create dispute: %w", err)
	}

	return item, nil
}

func (s *trackingService) ResolveDispute(ctx context.Context, adminID uuid.UUID, disputeID uuid.UUID, status, notes string) error {
	if status != "resolved_refund" && status != "resolved_payout" && status != "dismissed" {
		return errors.New("invalid dispute resolution status (must be resolved_refund, resolved_payout, or dismissed)")
	}

	return s.disputeRepo.ResolveDispute(ctx, disputeID, status, notes, adminID)
}

func (s *trackingService) ListDisputes(ctx context.Context, status string) ([]repository.DisputeItem, error) {
	return s.disputeRepo.ListDisputes(ctx, status)
}
