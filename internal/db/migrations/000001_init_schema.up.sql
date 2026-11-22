-- Spider Freight System - Core Database Schema
-- Leg 1: Schema Initialization & Verification

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";
CREATE EXTENSION IF NOT EXISTS "postgis";

-- 1. Users table (E.164 phone number, role-based access, DPDP Act 2023 consent)
CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    phone VARCHAR(20) NOT NULL UNIQUE,
    role VARCHAR(20) NOT NULL CHECK (role IN ('trader', 'provider', 'admin')),
    full_name VARCHAR(100),
    email VARCHAR(150),
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    dpdp_consent_at TIMESTAMPTZ,
    dpdp_consent_version VARCHAR(20),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_users_phone ON users(phone);
CREATE INDEX IF NOT EXISTS idx_users_role ON users(role);

-- 2. OTP Verifications table (Rate-limited phone authentication)
CREATE TABLE IF NOT EXISTS otp_verifications (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    phone VARCHAR(20) NOT NULL,
    otp_hash VARCHAR(255) NOT NULL,
    attempts INT NOT NULL DEFAULT 0,
    max_attempts INT NOT NULL DEFAULT 3,
    expires_at TIMESTAMPTZ NOT NULL,
    verified_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_otp_phone_expires ON otp_verifications(phone, expires_at);

-- 3. Providers KYC table (GSTIN, IEC, Fleet RC verification)
CREATE TABLE IF NOT EXISTS providers_kyc (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE UNIQUE,
    company_name VARCHAR(200) NOT NULL,
    gstin VARCHAR(15) NOT NULL,
    gstin_verified BOOLEAN NOT NULL DEFAULT FALSE,
    iec VARCHAR(10),
    iec_verified BOOLEAN NOT NULL DEFAULT FALSE,
    pan VARCHAR(10),
    fleet_size INT NOT NULL DEFAULT 1,
    status VARCHAR(20) NOT NULL DEFAULT 'draft' CHECK (status IN ('draft', 'pending_review', 'approved', 'rejected')),
    rejection_reason TEXT,
    reviewed_by UUID REFERENCES users(id),
    reviewed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_providers_kyc_status ON providers_kyc(status);
CREATE INDEX IF NOT EXISTS idx_providers_kyc_gstin ON providers_kyc(gstin);

-- 4. Provider KYC Documents (Transport permit, Fleet RC, Geotagged photos)
CREATE TABLE IF NOT EXISTS provider_documents (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    provider_kyc_id UUID NOT NULL REFERENCES providers_kyc(id) ON DELETE CASCADE,
    doc_type VARCHAR(50) NOT NULL CHECK (doc_type IN (
        'transport_permit',
        'fleet_rc',
        'container_front',
        'container_side',
        'container_doors',
        'gst_certificate',
        'iec_certificate',
        'other'
    )),
    file_url VARCHAR(500) NOT NULL,
    file_name VARCHAR(255) NOT NULL,
    file_size_bytes BIGINT NOT NULL,
    mime_type VARCHAR(100) NOT NULL,
    latitude DOUBLE PRECISION,
    longitude DOUBLE PRECISION,
    captured_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_provider_docs_kyc ON provider_documents(provider_kyc_id);

-- 5. Containers Table (Spatial PostGIS Points, Volume in CBM, Weight in KG, Pricing)
CREATE TABLE IF NOT EXISTS containers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    provider_id UUID NOT NULL REFERENCES users(id),
    iso_container_code VARCHAR(20) NOT NULL,
    transport_mode VARCHAR(20) NOT NULL CHECK (transport_mode IN ('rail', 'road', 'sea', 'air')),
    origin_title VARCHAR(150) NOT NULL,
    origin_location GEOMETRY(Point, 4326) NOT NULL,
    dest_title VARCHAR(150) NOT NULL,
    dest_location GEOMETRY(Point, 4326) NOT NULL,
    departure_time TIMESTAMPTZ NOT NULL,
    estimated_arrival_time TIMESTAMPTZ NOT NULL,
    total_cbm NUMERIC(8, 2) NOT NULL CHECK (total_cbm > 0),
    available_cbm NUMERIC(8, 2) NOT NULL CHECK (available_cbm >= 0 AND available_cbm <= total_cbm),
    total_weight_kg NUMERIC(10, 2) NOT NULL CHECK (total_weight_kg > 0),
    available_weight_kg NUMERIC(10, 2) NOT NULL CHECK (available_weight_kg >= 0 AND available_weight_kg <= total_weight_kg),
    rate_per_cbm_paise BIGINT NOT NULL CHECK (rate_per_cbm_paise >= 0),
    rate_per_kg_paise BIGINT NOT NULL DEFAULT 0 CHECK (rate_per_kg_paise >= 0),
    min_chargeable_cbm NUMERIC(6, 2) NOT NULL DEFAULT 0.1,
    status VARCHAR(20) NOT NULL DEFAULT 'scheduled' CHECK (status IN (
        'scheduled', 'boarding', 'in_transit', 'arrived', 'completed', 'cancelled'
    )),
    current_location GEOMETRY(Point, 4326),
    last_location_updated_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_containers_origin_loc ON containers USING GIST (origin_location);
CREATE INDEX IF NOT EXISTS idx_containers_dest_loc ON containers USING GIST (dest_location);
CREATE INDEX IF NOT EXISTS idx_containers_departure ON containers (departure_time);
CREATE INDEX IF NOT EXISTS idx_containers_status ON containers (status);

-- 6. Bookings Table (Single-transaction hold, chargeable weight/volume, dangerous goods, e-way bill)
CREATE TABLE IF NOT EXISTS bookings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    booking_ref VARCHAR(30) NOT NULL UNIQUE,
    container_id UUID NOT NULL REFERENCES containers(id),
    trader_id UUID NOT NULL REFERENCES users(id),
    status VARCHAR(20) NOT NULL CHECK (status IN (
        'held', 'confirmed', 'in_transit', 'delivered', 'disputed', 'cancelled', 'expired', 'refunded'
    )),
    consignment_length_cm NUMERIC(6, 1) NOT NULL,
    consignment_width_cm NUMERIC(6, 1) NOT NULL,
    consignment_height_cm NUMERIC(6, 1) NOT NULL,
    actual_weight_kg NUMERIC(10, 2) NOT NULL,
    computed_volume_cbm NUMERIC(8, 2) NOT NULL,
    chargeable_volume_cbm NUMERIC(8, 2) NOT NULL,
    chargeable_weight_kg NUMERIC(10, 2) NOT NULL,
    is_dangerous_goods BOOLEAN NOT NULL DEFAULT FALSE,
    cargo_description TEXT NOT NULL,
    eway_bill_ref VARCHAR(50),
    base_amount_paise BIGINT NOT NULL,
    platform_fee_paise BIGINT NOT NULL,
    gst_amount_paise BIGINT NOT NULL,
    total_amount_paise BIGINT NOT NULL,
    hold_expires_at TIMESTAMPTZ,
    confirmed_at TIMESTAMPTZ,
    delivered_at TIMESTAMPTZ,
    cancelled_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_bookings_container ON bookings(container_id);
CREATE INDEX IF NOT EXISTS idx_bookings_trader ON bookings(trader_id);
CREATE INDEX IF NOT EXISTS idx_bookings_status_hold_expires ON bookings(status, hold_expires_at) WHERE status = 'held';

-- 7. Payments & Escrow Table (Delayed payouts via Razorpay Route upon POD acceptance)
CREATE TABLE IF NOT EXISTS payments_escrow (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    booking_id UUID NOT NULL REFERENCES bookings(id) ON DELETE CASCADE UNIQUE,
    razorpay_order_id VARCHAR(100) NOT NULL UNIQUE,
    razorpay_payment_id VARCHAR(100) UNIQUE,
    razorpay_signature VARCHAR(255),
    amount_paise BIGINT NOT NULL,
    currency VARCHAR(5) NOT NULL DEFAULT 'INR',
    escrow_status VARCHAR(30) NOT NULL DEFAULT 'pending' CHECK (escrow_status IN (
        'pending', 'held', 'released_to_provider', 'refunded_to_trader', 'disputed_freeze'
    )),
    provider_route_transfer_id VARCHAR(100),
    provider_payout_amount_paise BIGINT NOT NULL,
    platform_commission_paise BIGINT NOT NULL,
    payout_released_at TIMESTAMPTZ,
    refund_id VARCHAR(100),
    refund_amount_paise BIGINT,
    refunded_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_payments_razorpay_order ON payments_escrow(razorpay_order_id);
CREATE INDEX IF NOT EXISTS idx_payments_razorpay_payment ON payments_escrow(razorpay_payment_id);
CREATE INDEX IF NOT EXISTS idx_payments_escrow_status ON payments_escrow(escrow_status);

-- 8. Processed Webhooks Table (Idempotent webhook reconciliation)
CREATE TABLE IF NOT EXISTS processed_webhooks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    event_id VARCHAR(100) NOT NULL UNIQUE,
    payment_id VARCHAR(100) NOT NULL,
    event_type VARCHAR(100) NOT NULL,
    payload JSONB NOT NULL,
    processed_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_webhooks_payment_id ON processed_webhooks(payment_id);

-- 9. Booking Documents Vault (Bill of Lading, Commercial Invoice, Packing List, E-way bill)
CREATE TABLE IF NOT EXISTS booking_documents (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    booking_id UUID NOT NULL REFERENCES bookings(id) ON DELETE CASCADE,
    doc_type VARCHAR(50) NOT NULL CHECK (doc_type IN (
        'bill_of_lading', 'commercial_invoice', 'packing_list', 'eway_bill', 'pod_photo', 'dispute_evidence'
    )),
    file_url VARCHAR(500) NOT NULL,
    file_name VARCHAR(255) NOT NULL,
    file_size_bytes BIGINT NOT NULL,
    mime_type VARCHAR(100) NOT NULL,
    uploaded_by UUID NOT NULL REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_booking_docs_booking ON booking_documents(booking_id);

-- 10. Delivery Proofs Table (POD photo, geocodes, recipient OTP confirmation)
CREATE TABLE IF NOT EXISTS delivery_proofs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    booking_id UUID NOT NULL REFERENCES bookings(id) ON DELETE CASCADE UNIQUE,
    pod_photo_url VARCHAR(500) NOT NULL,
    latitude DOUBLE PRECISION,
    longitude DOUBLE PRECISION,
    recipient_name VARCHAR(100) NOT NULL,
    recipient_phone VARCHAR(20) NOT NULL,
    recipient_signature_url VARCHAR(500),
    delivery_otp_hash VARCHAR(255),
    confirmed_by_trader BOOLEAN NOT NULL DEFAULT FALSE,
    disputed BOOLEAN NOT NULL DEFAULT FALSE,
    delivered_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 11. Disputes Table
CREATE TABLE IF NOT EXISTS disputes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    booking_id UUID NOT NULL REFERENCES bookings(id) ON DELETE CASCADE,
    raised_by UUID NOT NULL REFERENCES users(id),
    reason VARCHAR(100) NOT NULL,
    description TEXT NOT NULL,
    evidence_urls JSONB NOT NULL DEFAULT '[]'::jsonb,
    status VARCHAR(30) NOT NULL DEFAULT 'open' CHECK (status IN (
        'open', 'under_investigation', 'resolved_refund', 'resolved_payout', 'dismissed'
    )),
    resolution_notes TEXT,
    resolved_by UUID REFERENCES users(id),
    resolved_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_disputes_booking ON disputes(booking_id);
CREATE INDEX IF NOT EXISTS idx_disputes_status ON disputes(status);

-- 12. Chat Messages Table (Contextual chat scoped to bookings)
CREATE TABLE IF NOT EXISTS chat_messages (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    booking_id UUID NOT NULL REFERENCES bookings(id) ON DELETE CASCADE,
    sender_id UUID NOT NULL REFERENCES users(id),
    content TEXT NOT NULL,
    attachment_url VARCHAR(500),
    is_read BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_chat_booking_created ON chat_messages(booking_id, created_at);

-- 13. Transactional Outbox Table (Eventual consistency for push, WhatsApp, payouts)
CREATE TABLE IF NOT EXISTS outbox (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    event_type VARCHAR(100) NOT NULL,
    aggregate_type VARCHAR(50) NOT NULL,
    aggregate_id UUID NOT NULL,
    payload JSONB NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'pending' CHECK (status IN (
        'pending', 'processing', 'completed', 'failed'
    )),
    retry_count INT NOT NULL DEFAULT 0,
    max_retries INT NOT NULL DEFAULT 5,
    last_error TEXT,
    next_attempt_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    processed_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_outbox_pending ON outbox(status, next_attempt_at) WHERE status = 'pending';

-- 14. Immutable Admin Audit Logs Table
CREATE TABLE IF NOT EXISTS audit_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    admin_id UUID REFERENCES users(id),
    action VARCHAR(100) NOT NULL,
    target_type VARCHAR(50) NOT NULL,
    target_id UUID NOT NULL,
    changes JSONB NOT NULL,
    ip_address VARCHAR(50),
    user_agent TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_audit_logs_target ON audit_logs(target_type, target_id);
CREATE INDEX IF NOT EXISTS idx_audit_logs_admin ON audit_logs(admin_id);
CREATE INDEX IF NOT EXISTS idx_audit_logs_created ON audit_logs(created_at);
