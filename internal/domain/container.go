package domain

import (
	"time"

	"github.com/google/uuid"
)

type TransportMode string

const (
	TransportModeRail TransportMode = "rail"
	TransportModeRoad TransportMode = "road"
	TransportModeSea  TransportMode = "sea"
	TransportModeAir  TransportMode = "air"
)

type ContainerStatus string

const (
	ContainerStatusScheduled ContainerStatus = "scheduled"
	ContainerStatusBoarding  ContainerStatus = "boarding"
	ContainerStatusInTransit ContainerStatus = "in_transit"
	ContainerStatusArrived   ContainerStatus = "arrived"
	ContainerStatusCompleted ContainerStatus = "completed"
	ContainerStatusCancelled ContainerStatus = "cancelled"
)

type LocationPoint struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

type Container struct {
	ID                    uuid.UUID       `json:"id"`
	ProviderID            uuid.UUID       `json:"provider_id"`
	ISOContainerCode      string          `json:"iso_container_code"`
	TransportMode         TransportMode   `json:"transport_mode"`
	OriginTitle           string          `json:"origin_title"`
	OriginLocation        LocationPoint   `json:"origin_location"`
	DestTitle             string          `json:"dest_title"`
	DestLocation          LocationPoint   `json:"dest_location"`
	DepartureTime         time.Time       `json:"departure_time"`
	EstimatedArrivalTime  time.Time       `json:"estimated_arrival_time"`
	TotalCBM              float64         `json:"total_cbm"`
	AvailableCBM          float64         `json:"available_cbm"`
	TotalWeightKG         float64         `json:"total_weight_kg"`
	AvailableWeightKG     float64         `json:"available_weight_kg"`
	RatePerCBMPaise       int64           `json:"rate_per_cbm_paise"`
	RatePerKGPaise        int64           `json:"rate_per_kg_paise"`
	MinChargeableCBM      float64         `json:"min_chargeable_cbm"`
	Status                ContainerStatus `json:"status"`
	CurrentLocation       *LocationPoint  `json:"current_location,omitempty"`
	LastLocationUpdatedAt *time.Time      `json:"last_location_updated_at,omitempty"`
	CreatedAt             time.Time       `json:"created_at"`
	UpdatedAt             time.Time       `json:"updated_at"`
}
