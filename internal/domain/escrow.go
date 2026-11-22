package domain

import (
	"time"

	"github.com/google/uuid"
)

type EscrowStatus string

const (
	EscrowStatusPending            EscrowStatus = "pending"
	EscrowStatusHeld               EscrowStatus = "held"
	EscrowStatusReleasedToProvider EscrowStatus = "released_to_provider"
	EscrowStatusRefundedToTrader   EscrowStatus = "refunded_to_trader"
	EscrowStatusDisputedFreeze     EscrowStatus = "disputed_freeze"
)

type PaymentEscrow struct {
	ID                       uuid.UUID    `json:"id"`
	BookingID                uuid.UUID    `json:"booking_id"`
	RazorpayOrderID          string       `json:"razorpay_order_id"`
	RazorpayPaymentID        string       `json:"razorpay_payment_id,omitempty"`
	RazorpaySignature        string       `json:"razorpay_signature,omitempty"`
	AmountPaise              int64        `json:"amount_paise"`
	Currency                 string       `json:"currency"`
	EscrowStatus             EscrowStatus `json:"escrow_status"`
	ProviderRouteTransferID  string       `json:"provider_route_transfer_id,omitempty"`
	ProviderPayoutAmountPaise int64       `json:"provider_payout_amount_paise"`
	PlatformCommissionPaise  int64        `json:"platform_commission_paise"`
	PayoutReleasedAt         *time.Time   `json:"payout_released_at,omitempty"`
	RefundID                 string       `json:"refund_id,omitempty"`
	RefundAmountPaise        *int64       `json:"refund_amount_paise,omitempty"`
	RefundedAt               *time.Time   `json:"refunded_at,omitempty"`
	CreatedAt                time.Time    `json:"created_at"`
	UpdatedAt                time.Time    `json:"updated_at"`
}
