package service

import (
	"context"
	"errors"
	"fmt"
	"math"
	"regexp"
	"strings"
	"time"

	"github.com/Aldrinsui/Spider/internal/domain"
	"github.com/Aldrinsui/Spider/internal/repository"
	"github.com/google/uuid"
)

var isoContainerRegex = regexp.MustCompile(`^[A-Z]{4}\s?[0-9]{6,7}$`)

type QuoteRequest struct {
	ContainerID      uuid.UUID `json:"container_id"`
	LengthCM         float64   `json:"length_cm"`
	WidthCM          float64   `json:"width_cm"`
	HeightCM         float64   `json:"height_cm"`
	ActualWeightKG   float64   `json:"actual_weight_kg"`
	IsDangerousGoods bool      `json:"is_dangerous_goods"`
}

type QuoteCalculation struct {
	ComputedVolumeCBM   float64 `json:"computed_volume_cbm"`
	ChargeableVolumeCBM float64 `json:"chargeable_volume_cbm"`
	ActualWeightKG      float64 `json:"actual_weight_kg"`
	ChargeableWeightKG  float64 `json:"chargeable_weight_kg"`
	RatePerCBMPaise     int64   `json:"rate_per_cbm_paise"`
	BaseAmountPaise     int64   `json:"base_amount_paise"`
	PlatformFeePaise    int64   `json:"platform_fee_paise"`
	GSTAmountPaise      int64   `json:"gst_amount_paise"`
	TotalAmountPaise    int64   `json:"total_amount_paise"`
	Currency            string  `json:"currency"`
}

type ContainerService interface {
	CreateContainer(ctx context.Context, providerID uuid.UUID, req domain.Container) (*domain.Container, error)
	GetContainer(ctx context.Context, id uuid.UUID) (*domain.Container, error)
	SearchContainers(ctx context.Context, filter repository.ContainerSearchFilter) ([]domain.Container, error)
	ListProviderContainers(ctx context.Context, providerID uuid.UUID) ([]domain.Container, error)
	CalculateQuote(ctx context.Context, req QuoteRequest) (*QuoteCalculation, error)
}

type containerService struct {
	containerRepo repository.ContainerRepository
	kycRepo       repository.KYCRepository
}

func NewContainerService(containerRepo repository.ContainerRepository, kycRepo repository.KYCRepository) ContainerService {
	return &containerService{containerRepo: containerRepo, kycRepo: kycRepo}
}

func (s *containerService) CreateContainer(ctx context.Context, providerID uuid.UUID, c domain.Container) (*domain.Container, error) {
	// Verify that provider is approved before allowing container listing
	if s.kycRepo != nil {
		kyc, err := s.kycRepo.GetKYCByUserID(ctx, providerID)
		if err != nil {
			return nil, fmt.Errorf("error verifying provider KYC: %w", err)
		}
		if kyc == nil || kyc.Status != domain.KYCStatusApproved {
			return nil, errors.New("only approved logistics providers can list containers. Please complete KYC verification.")
		}
	}

	c.ISOContainerCode = strings.ToUpper(strings.TrimSpace(c.ISOContainerCode))
	if !isoContainerRegex.MatchString(c.ISOContainerCode) {
		return nil, errors.New("invalid ISO container code format (e.g. MSKU 482913 or TCLU 110452)")
	}

	if c.TotalCBM <= 0 {
		return nil, errors.New("total CBM capacity must be greater than zero")
	}
	if c.TotalWeightKG <= 0 {
		return nil, errors.New("total payload weight in kg must be greater than zero")
	}
	if c.RatePerCBMPaise <= 0 {
		return nil, errors.New("rate per CBM must be greater than zero")
	}

	if c.DepartureTime.Before(time.Now()) {
		return nil, errors.New("departure time must be in the future")
	}
	if c.EstimatedArrivalTime.Before(c.DepartureTime) {
		return nil, errors.New("estimated arrival time must be after departure time")
	}

	c.ProviderID = providerID
	c.AvailableCBM = c.TotalCBM
	c.AvailableWeightKG = c.TotalWeightKG

	if err := s.containerRepo.CreateContainer(ctx, &c); err != nil {
		return nil, fmt.Errorf("failed to create container: %w", err)
	}

	return &c, nil
}

func (s *containerService) GetContainer(ctx context.Context, id uuid.UUID) (*domain.Container, error) {
	return s.containerRepo.GetContainerByID(ctx, id)
}

func (s *containerService) SearchContainers(ctx context.Context, filter repository.ContainerSearchFilter) ([]domain.Container, error) {
	return s.containerRepo.SearchContainers(ctx, filter)
}

func (s *containerService) ListProviderContainers(ctx context.Context, providerID uuid.UUID) ([]domain.Container, error) {
	return s.containerRepo.ListByProvider(ctx, providerID)
}

func (s *containerService) CalculateQuote(ctx context.Context, req QuoteRequest) (*QuoteCalculation, error) {
	if req.LengthCM <= 0 || req.WidthCM <= 0 || req.HeightCM <= 0 {
		return nil, errors.New("consignment dimensions (L, W, H in cm) must be greater than zero")
	}
	if req.ActualWeightKG <= 0 {
		return nil, errors.New("consignment weight in kg must be greater than zero")
	}

	container, err := s.containerRepo.GetContainerByID(ctx, req.ContainerID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch container: %w", err)
	}
	if container == nil {
		return nil, errors.New("container not found")
	}

	// 1. Compute consignment volume in CBM (m^3)
	// Volume = (L * W * H in cm) / 1,000,000
	rawVolumeCBM := (req.LengthCM * req.WidthCM * req.HeightCM) / 1000000.0
	computedVolumeCBM := math.Round(rawVolumeCBM*100) / 100 // 2 decimal precision

	// 2. Chargeable volume: max(computedVolume, min_chargeable_cbm)
	chargeableVolumeCBM := computedVolumeCBM
	if chargeableVolumeCBM < container.MinChargeableCBM {
		chargeableVolumeCBM = container.MinChargeableCBM
	}

	// 3. Volumetric weight check (standard 1 CBM = 167 kg for road/rail/air freight)
	volumetricWeightKG := computedVolumeCBM * 167.0
	chargeableWeightKG := req.ActualWeightKG
	if volumetricWeightKG > chargeableWeightKG {
		chargeableWeightKG = volumetricWeightKG
	}

	// 4. Base amount calculation in paise
	// Base = chargeableVolumeCBM * ratePerCBM
	baseAmountPaise := int64(math.Round(chargeableVolumeCBM * float64(container.RatePerCBMPaise)))

	// 5. Platform fee = 5% of base amount
	platformFeePaise := int64(math.Round(float64(baseAmountPaise) * 0.05))

	// 6. GST = 18% on platform fee + 5% on freight base amount (Indian GST composite freight rule)
	// Base transport GST (5%):
	freightGST := int64(math.Round(float64(baseAmountPaise) * 0.05))
	// Platform service GST (18%):
	platformGST := int64(math.Round(float64(platformFeePaise) * 0.18))
	gstAmountPaise := freightGST + platformGST

	// 7. Total amount = Base + Platform Fee + GST
	totalAmountPaise := baseAmountPaise + platformFeePaise + gstAmountPaise

	return &QuoteCalculation{
		ComputedVolumeCBM:   computedVolumeCBM,
		ChargeableVolumeCBM: chargeableVolumeCBM,
		ActualWeightKG:      req.ActualWeightKG,
		ChargeableWeightKG:  chargeableWeightKG,
		RatePerCBMPaise:     container.RatePerCBMPaise,
		BaseAmountPaise:     baseAmountPaise,
		PlatformFeePaise:    platformFeePaise,
		GSTAmountPaise:      gstAmountPaise,
		TotalAmountPaise:    totalAmountPaise,
		Currency:            "INR",
	}, nil
}
