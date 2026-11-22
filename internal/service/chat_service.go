package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/Aldrinsui/Spider/internal/domain"
	"github.com/Aldrinsui/Spider/internal/repository"
	"github.com/google/uuid"
)

type ChatService interface {
	SendMessage(ctx context.Context, senderID uuid.UUID, bookingID uuid.UUID, content, attachmentURL string) (*domain.ChatMessage, error)
	GetMessages(ctx context.Context, userID uuid.UUID, bookingID uuid.UUID, limit int, before *time.Time) ([]domain.ChatMessage, error)
}

type chatService struct {
	chatRepo      repository.ChatRepository
	bookingRepo   repository.BookingRepository
	containerRepo repository.ContainerRepository
}

func NewChatService(chatRepo repository.ChatRepository, bookingRepo repository.BookingRepository, containerRepo repository.ContainerRepository) ChatService {
	return &chatService{
		chatRepo:      chatRepo,
		bookingRepo:   bookingRepo,
		containerRepo: containerRepo,
	}
}

func (s *chatService) SendMessage(ctx context.Context, senderID uuid.UUID, bookingID uuid.UUID, content, attachmentURL string) (*domain.ChatMessage, error) {
	content = strings.TrimSpace(content)
	if content == "" && attachmentURL == "" {
		return nil, errors.New("chat message must contain text or attachment")
	}

	// Verify sender is participant in booking
	booking, err := s.bookingRepo.GetBookingByID(ctx, bookingID)
	if err != nil || booking == nil {
		return nil, errors.New("booking not found")
	}

	container, err := s.containerRepo.GetContainerByID(ctx, booking.ContainerID)
	if err != nil || container == nil {
		return nil, errors.New("container not found")
	}

	isTrader := booking.TraderID == senderID
	isProvider := container.ProviderID == senderID

	if !isTrader && !isProvider {
		return nil, errors.New("forbidden: you are not a participant in this consignment")
	}

	msg := &domain.ChatMessage{
		BookingID:     bookingID,
		SenderID:      senderID,
		Content:       content,
		AttachmentURL: attachmentURL,
		IsRead:        false,
	}

	if err := s.chatRepo.SendMessage(ctx, msg); err != nil {
		return nil, fmt.Errorf("failed to send message: %w", err)
	}

	return msg, nil
}

func (s *chatService) GetMessages(ctx context.Context, userID uuid.UUID, bookingID uuid.UUID, limit int, before *time.Time) ([]domain.ChatMessage, error) {
	booking, err := s.bookingRepo.GetBookingByID(ctx, bookingID)
	if err != nil || booking == nil {
		return nil, errors.New("booking not found")
	}

	container, err := s.containerRepo.GetContainerByID(ctx, booking.ContainerID)
	if err != nil || container == nil {
		return nil, errors.New("container not found")
	}

	if booking.TraderID != userID && container.ProviderID != userID {
		return nil, errors.New("forbidden: not a participant")
	}

	// Mark messages read by viewer first
	_ = s.chatRepo.MarkMessagesAsRead(ctx, bookingID, userID)

	return s.chatRepo.ListMessages(ctx, bookingID, limit, before)
}
