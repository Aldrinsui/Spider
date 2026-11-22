# Spider Freight — Cargo Space Booking & Fleet Platform

A high-performance, fault-tolerant freight booking and transport allocation platform built with **Go**, **PostgreSQL with PostGIS**, and **Flutter**, engineered specifically for Indian intermodal logistics (Rail, Road, Sea, Air).

---

## 🌟 Key Architecture Corrections & Features

1. **No Offline Double-Booking**: 
   - Search, space reservations (holds), and payment processing strictly require an active server connection.
   - Drift/local cache is strictly limited to offline read caching (viewing saved routes, confirmed consignment docs) and queued messages.
2. **Atomic Single-Transaction Holds**:
   - Capacity verification, container decrement (`available_cbm` and `available_weight_kg`), booking insertion in `held` status, and outbox event creation commit within a single atomic database transaction.
   - Guarded by explicit `CHECK (available_cbm >= 0 AND available_weight_kg >= 0)` and row-level locks (`SELECT ... FOR UPDATE`).
3. **Automated Expiry Reclaimer**:
   - Background reclaimer process polls `bookings WHERE status = 'held' AND hold_expires_at < NOW() FOR UPDATE SKIP LOCKED`.
   - Automatically restores container capacity and marks the hold `expired`.
4. **Razorpay Escrow & Late Webhook Reconciliation**:
   - Delayed transfers via Razorpay Route hold provider payouts in escrow until proof of delivery (POD).
   - Idempotent webhook handling with deduplication on `razorpay_payment_id`.
   - If a payment webhook arrives after a hold has expired, the server checks if container space is still available: confirms the shipment if free, otherwise triggers an automated refund.
5. **Indian Freight Domain Specifics**:
   - Chargeable weight vs volumetric weight ($L \times W \times H\text{ cm} \rightarrow \text{CBM}$) with standard freight ratios.
   - Mandatory Dangerous Goods declaration and automated prohibited cargo screening.
   - GSTIN (15-character) & IEC verification with GST composite calculation.
   - E-way bill reference tracking for road shipments.
   - Document vault per consignment (Bill of Lading, Commercial Invoice, Packing List).
   - Proof of Delivery (POD) photo upload, recipient verification, and geocoded delivery coordinates.
   - DPDP Act 2023 consent capture and immutable admin audit trails.

---

## 🌿 Git Branch Structure (`leg` Convention)

| Branch | Milestone / Focus | Status |
|---|---|---|
| [`leg1-schema-and-auth`](https://github.com/Aldrinsui/Spider/tree/leg1-schema-and-auth) | Revised PostgreSQL Schema, OTP phone auth, Provider KYC, and Admin approval | ✅ Merged & Pushed |
| [`leg2-containers-and-spatial-search`](https://github.com/Aldrinsui/Spider/tree/leg2-containers-and-spatial-search) | PostGIS corridor search, ISO container registry, volumetric quote engine | ✅ Merged & Pushed |
| [`leg3-atomic-booking-holds-and-reclaimer`](https://github.com/Aldrinsui/Spider/tree/leg3-atomic-booking-holds-and-reclaimer) | Atomic transaction holds (10-min window), Transactional Outbox, Reclaimer worker | ✅ Merged & Pushed |
| [`leg4-razorpay-escrow-and-webhooks`](https://github.com/Aldrinsui/Spider/tree/leg4-razorpay-escrow-and-webhooks) | Razorpay webhook deduplication, late payment reconciliation, delayed escrow payout | ✅ Merged & Pushed |
| [`leg5-contextual-chat-and-notifications`](https://github.com/Aldrinsui/Spider/tree/leg5-contextual-chat-and-notifications) | Scoped consignment chat, PostgreSQL `LISTEN/NOTIFY`, Outbox push dispatcher | ✅ Merged & Pushed |
| [`leg6-tracking-documents-disputes-and-flutter`](https://github.com/Aldrinsui/Spider/tree/leg6-tracking-documents-disputes-and-flutter) | GPS tracking, POD photo escrow release, dispute arbitration, Flutter UI | ✅ Merged & Pushed |
| `main` | Production unified codebase | ✅ Up to date |

---

## 🚀 Quickstart & Setup

### 1. Run with Docker Compose
```bash
docker-compose up -d
```
Spins up PostgreSQL 16 with PostGIS 3.4 and applies all migrations automatically.

### 2. Run Backend Unit & Integration Tests
```bash
go test -v ./test/...
```
All 10 test suites covering Auth, KYC, Containers, Holds, Reclaimer, Payments, Webhooks, Chat, Tracking, and Disputes will run.

### 3. Start Backend Services
```bash
# Start REST API server (port 8080)
go run cmd/api/main.go

# Start Background Expiry Reclaimer
go run cmd/reclaimer/main.go
```

### 4. Run Flutter Mobile App
```bash
cd mobile
flutter pub get
flutter run
```

---

## 📱 Mobile App Screens (Wireframes Implemented)

- **Map Search (`MapSearchScreen`)**:
  - Interactive dark map canvas with corridor route lines.
  - Route options: Rail (MSKU 482913 · Thu 06:00 · 18.6 / 30 CBM free · Rs 740/CBM) and Road (TCLU 110452 · Fri 14:30 · 3.4 / 14 CBM free · Rs 610/CBM).
- **Container & Book (`ContainerAndBookScreen`)**:
  - Consignment dimension calculator ($120 \times 80 \times 90\text{ cm} \rightarrow 0.86\text{ CBM}$).
  - Live quote calculation: Rate Rs 740/CBM, Total before GST Rs 636.40.
  - 10-minute hold reservation button with countdown timer.
- **Provider Fleet Verification (`ProviderVerifyScreen`)**:
  - Transport permit upload, Fleet RC upload, and 3 container photos (Front, Side, Doors) with geotagging.
- **Admin Review Queue (`AdminReviewScreen`)**:
  - Document verification inspection, GSTIN status, Approve / Reject actions with immutable audit logs.
