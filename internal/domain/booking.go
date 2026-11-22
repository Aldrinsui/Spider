package domain

import (
	"time"

	"github.com/google/uuid"
)

type BookingStatus string

const (
	BookingStatusHeld       BookingStatus = "held"
	BookingStatusConfirmed  BookingStatus = "confirmed"
	BookingStatusInTransit  BookingStatus = "in_transit"
	BookingStatusDelivered  BookingStatus = "delivered"
	BookingStatusDisputed   BookingStatus = "disputed"
	BookingStatusCancelled  BookingStatus = "cancelled"
	BookingStatusExpired    BookingStatus = "expired"
	BookingStatusRefunded   BookingStatus = "refunded"
)

type Booking struct {
	ID                   uuid.UUID     `json:"id"`
	BookingRef           string        `json:"booking_ref"`
	ContainerID          uuid.UUID     `json:"container_id"`
	TraderID             uuid.UUID     `json:"trader_id"`
	Status               BookingStatus `json:"status"`
	ConsignmentLengthCM  float64       `json:"consignment_length_cm"`
	ConsignmentWidthCM   float64       `json:"consignment_width_cm"`
	ConsignmentHeightCM  float64       `json:"consignment_height_cm"`
	ActualWeightKG       float64       `json:"actual_weight_kg"`
	ComputedVolumeCBM    float64       `json:"computed_volume_cbm"`
	ChargeableVolumeCBM  float64       `json:"chargeable_volume_cbm"`
	ChargeableWeightKG   float64       `json:"chargeable_weight_kg"`
	IsDangerousGoods     bool          `json:"is_dangerous_goods"`
	CargoDescription     string        `json:"cargo_description"`
	EWayBillRef          string        `json:"eway_bill_ref,omitempty"`
	BaseAmountPaise      int64         `json:"base_amount_paise"`
	PlatformFeePaise     int64         `json:"platform_fee_paise"`
	GSTAmountPaise       int64         `json:"gst_amount_paise"`
	TotalAmountPaise     int64         `json:"total_amount_paise"`
	HoldExpiresAt        *time.Time    `json:"hold_expires_at,omitempty"`
	ConfirmedAt          *time.Time    `json:"confirmed_at,omitempty"`
	DeliveredAt          *time.Time    `json:"delivered_at,omitempty"`
	CancelledAt          *time.Time    `json:"cancelled_at,omitempty"`
	CreatedAt            time.Time     `json:"created_at"`
	UpdatedAt            time.Time     `json:"updated_at"`
}
