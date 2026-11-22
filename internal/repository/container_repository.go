package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/Aldrinsui/Spider/internal/domain"
	"github.com/google/uuid"
)

type ContainerSearchFilter struct {
	OriginLat     *float64
	OriginLng     *float64
	DestLat       *float64
	DestLng       *float64
	RadiusKM      float64
	MinCBM        float64
	MinWeightKG   float64
	TransportMode string
	SortBy        string // "departure", "rate", "available_space"
	Limit         int
	Offset        int
}

type ContainerRepository interface {
	CreateContainer(ctx context.Context, c *domain.Container) error
	GetContainerByID(ctx context.Context, id uuid.UUID) (*domain.Container, error)
	UpdateContainer(ctx context.Context, c *domain.Container) error
	SearchContainers(ctx context.Context, filter ContainerSearchFilter) ([]domain.Container, error)
	ListByProvider(ctx context.Context, providerID uuid.UUID) ([]domain.Container, error)
}

type sqlContainerRepository struct {
	db *sql.DB
}

func NewContainerRepository(db *sql.DB) ContainerRepository {
	return &sqlContainerRepository{db: db}
}

func (r *sqlContainerRepository) CreateContainer(ctx context.Context, c *domain.Container) error {
	query := `
		INSERT INTO containers (
			id, provider_id, iso_container_code, transport_mode,
			origin_title, origin_location,
			dest_title, dest_location,
			departure_time, estimated_arrival_time,
			total_cbm, available_cbm,
			total_weight_kg, available_weight_kg,
			rate_per_cbm_paise, rate_per_kg_paise,
			min_chargeable_cbm, status,
			created_at, updated_at
		) VALUES (
			$1, $2, $3, $4,
			$5, ST_SetSRID(ST_MakePoint($6, $7), 4326),
			$8, ST_SetSRID(ST_MakePoint($9, $10), 4326),
			$11, $12,
			$13, $14,
			$15, $16,
			$17, $18,
			$19, $20,
			NOW(), NOW()
		)
		RETURNING id, created_at, updated_at
	`
	if c.ID == uuid.Nil {
		c.ID = uuid.New()
	}
	if c.AvailableCBM == 0 {
		c.AvailableCBM = c.TotalCBM
	}
	if c.AvailableWeightKG == 0 {
		c.AvailableWeightKG = c.TotalWeightKG
	}
	if c.Status == "" {
		c.Status = domain.ContainerStatusScheduled
	}
	if c.MinChargeableCBM <= 0 {
		c.MinChargeableCBM = 0.1
	}

	return r.db.QueryRowContext(ctx, query,
		c.ID,
		c.ProviderID,
		c.ISOContainerCode,
		c.TransportMode,
		c.OriginTitle,
		c.OriginLocation.Longitude,
		c.OriginLocation.Latitude,
		c.DestTitle,
		c.DestLocation.Longitude,
		c.DestLocation.Latitude,
		c.DepartureTime,
		c.EstimatedArrivalTime,
		c.TotalCBM,
		c.AvailableCBM,
		c.TotalWeightKG,
		c.AvailableWeightKG,
		c.RatePerCBMPaise,
		c.RatePerKGPaise,
		c.MinChargeableCBM,
		c.Status,
	).Scan(&c.ID, &c.CreatedAt, &c.UpdatedAt)
}

func (r *sqlContainerRepository) GetContainerByID(ctx context.Context, id uuid.UUID) (*domain.Container, error) {
	query := `
		SELECT 
			id, provider_id, iso_container_code, transport_mode,
			origin_title, ST_Y(origin_location::geometry) as origin_lat, ST_X(origin_location::geometry) as origin_lng,
			dest_title, ST_Y(dest_location::geometry) as dest_lat, ST_X(dest_location::geometry) as dest_lng,
			departure_time, estimated_arrival_time,
			total_cbm, available_cbm,
			total_weight_kg, available_weight_kg,
			rate_per_cbm_paise, rate_per_kg_paise,
			min_chargeable_cbm, status,
			created_at, updated_at
		FROM containers
		WHERE id = $1
	`
	c := &domain.Container{}
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&c.ID,
		&c.ProviderID,
		&c.ISOContainerCode,
		&c.TransportMode,
		&c.OriginTitle,
		&c.OriginLocation.Latitude,
		&c.OriginLocation.Longitude,
		&c.DestTitle,
		&c.DestLocation.Latitude,
		&c.DestLocation.Longitude,
		&c.DepartureTime,
		&c.EstimatedArrivalTime,
		&c.TotalCBM,
		&c.AvailableCBM,
		&c.TotalWeightKG,
		&c.AvailableWeightKG,
		&c.RatePerCBMPaise,
		&c.RatePerKGPaise,
		&c.MinChargeableCBM,
		&c.Status,
		&c.CreatedAt,
		&c.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return c, nil
}

func (r *sqlContainerRepository) UpdateContainer(ctx context.Context, c *domain.Container) error {
	query := `
		UPDATE containers
		SET iso_container_code = $2,
		    departure_time = $3,
		    estimated_arrival_time = $4,
		    rate_per_cbm_paise = $5,
		    rate_per_kg_paise = $6,
		    status = $7,
		    updated_at = NOW()
		WHERE id = $1
	`
	_, err := r.db.ExecContext(ctx, query,
		c.ID,
		c.ISOContainerCode,
		c.DepartureTime,
		c.EstimatedArrivalTime,
		c.RatePerCBMPaise,
		c.RatePerKGPaise,
		c.Status,
	)
	return err
}

func (r *sqlContainerRepository) SearchContainers(ctx context.Context, filter ContainerSearchFilter) ([]domain.Container, error) {
	var conditions []string
	var args []interface{}
	argIdx := 1

	conditions = append(conditions, "status = 'scheduled'")
	conditions = append(conditions, "departure_time > NOW()")

	if filter.MinCBM > 0 {
		conditions = append(conditions, fmt.Sprintf("available_cbm >= $%d", argIdx))
		args = append(args, filter.MinCBM)
		argIdx++
	}

	if filter.MinWeightKG > 0 {
		conditions = append(conditions, fmt.Sprintf("available_weight_kg >= $%d", argIdx))
		args = append(args, filter.MinWeightKG)
		argIdx++
	}

	if filter.TransportMode != "" {
		conditions = append(conditions, fmt.Sprintf("transport_mode = $%d", argIdx))
		args = append(args, filter.TransportMode)
		argIdx++
	}

	radiusMeters := filter.RadiusKM * 1000
	if radiusMeters <= 0 {
		radiusMeters = 50000 // default 50km radius
	}

	// Spatial filtering on Origin with PostGIS ST_DWithin
	if filter.OriginLat != nil && filter.OriginLng != nil {
		conditions = append(conditions, fmt.Sprintf("ST_DWithin(origin_location, ST_SetSRID(ST_MakePoint($%d, $%d), 4326)::geography, $%d)", argIdx, argIdx+1, argIdx+2))
		args = append(args, *filter.OriginLng, *filter.OriginLat, radiusMeters)
		argIdx += 3
	}

	// Spatial filtering on Destination with PostGIS ST_DWithin
	if filter.DestLat != nil && filter.DestLng != nil {
		conditions = append(conditions, fmt.Sprintf("ST_DWithin(dest_location, ST_SetSRID(ST_MakePoint($%d, $%d), 4326)::geography, $%d)", argIdx, argIdx+1, argIdx+2))
		args = append(args, *filter.DestLng, *filter.DestLat, radiusMeters)
		argIdx += 3
	}

	whereClause := "WHERE " + strings.Join(conditions, " AND ")

	orderClause := "ORDER BY departure_time ASC"
	switch filter.SortBy {
	case "rate":
		orderClause = "ORDER BY rate_per_cbm_paise ASC"
	case "available_space":
		orderClause = "ORDER BY available_cbm DESC"
	}

	limit := filter.Limit
	if limit <= 0 || limit > 50 {
		limit = 20
	}
	offset := filter.Offset
	if offset < 0 {
		offset = 0
	}

	query := fmt.Sprintf(`
		SELECT 
			id, provider_id, iso_container_code, transport_mode,
			origin_title, ST_Y(origin_location::geometry) as origin_lat, ST_X(origin_location::geometry) as origin_lng,
			dest_title, ST_Y(dest_location::geometry) as dest_lat, ST_X(dest_location::geometry) as dest_lng,
			departure_time, estimated_arrival_time,
			total_cbm, available_cbm,
			total_weight_kg, available_weight_kg,
			rate_per_cbm_paise, rate_per_kg_paise,
			min_chargeable_cbm, status,
			created_at, updated_at
		FROM containers
		%s
		%s
		LIMIT $%d OFFSET $%d
	`, whereClause, orderClause, argIdx, argIdx+1)

	args = append(args, limit, offset)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("search query failed: %w", err)
	}
	defer rows.Close()

	var containers []domain.Container
	for rows.Next() {
		var c domain.Container
		if err := rows.Scan(
			&c.ID,
			&c.ProviderID,
			&c.ISOContainerCode,
			&c.TransportMode,
			&c.OriginTitle,
			&c.OriginLocation.Latitude,
			&c.OriginLocation.Longitude,
			&c.DestTitle,
			&c.DestLocation.Latitude,
			&c.DestLocation.Longitude,
			&c.DepartureTime,
			&c.EstimatedArrivalTime,
			&c.TotalCBM,
			&c.AvailableCBM,
			&c.TotalWeightKG,
			&c.AvailableWeightKG,
			&c.RatePerCBMPaise,
			&c.RatePerKGPaise,
			&c.MinChargeableCBM,
			&c.Status,
			&c.CreatedAt,
			&c.UpdatedAt,
		); err != nil {
			return nil, err
		}
		containers = append(containers, c)
	}

	return containers, nil
}

func (r *sqlContainerRepository) ListByProvider(ctx context.Context, providerID uuid.UUID) ([]domain.Container, error) {
	query := `
		SELECT 
			id, provider_id, iso_container_code, transport_mode,
			origin_title, ST_Y(origin_location::geometry) as origin_lat, ST_X(origin_location::geometry) as origin_lng,
			dest_title, ST_Y(dest_location::geometry) as dest_lat, ST_X(dest_location::geometry) as dest_lng,
			departure_time, estimated_arrival_time,
			total_cbm, available_cbm,
			total_weight_kg, available_weight_kg,
			rate_per_cbm_paise, rate_per_kg_paise,
			min_chargeable_cbm, status,
			created_at, updated_at
		FROM containers
		WHERE provider_id = $1
		ORDER BY created_at DESC
	`
	rows, err := r.db.QueryContext(ctx, query, providerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var containers []domain.Container
	for rows.Next() {
		var c domain.Container
		if err := rows.Scan(
			&c.ID,
			&c.ProviderID,
			&c.ISOContainerCode,
			&c.TransportMode,
			&c.OriginTitle,
			&c.OriginLocation.Latitude,
			&c.OriginLocation.Longitude,
			&c.DestTitle,
			&c.DestLocation.Latitude,
			&c.DestLocation.Longitude,
			&c.DepartureTime,
			&c.EstimatedArrivalTime,
			&c.TotalCBM,
			&c.AvailableCBM,
			&c.TotalWeightKG,
			&c.AvailableWeightKG,
			&c.RatePerCBMPaise,
			&c.RatePerKGPaise,
			&c.MinChargeableCBM,
			&c.Status,
			&c.CreatedAt,
			&c.UpdatedAt,
		); err != nil {
			return nil, err
		}
		containers = append(containers, c)
	}

	return containers, nil
}
