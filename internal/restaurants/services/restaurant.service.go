package services

import (
	"context"
	"time"

	"github.com/alibaba0010/postgres-api/internal/common/errors"
	"github.com/alibaba0010/postgres-api/internal/common/logger"
	"github.com/alibaba0010/postgres-api/internal/database"
	"github.com/alibaba0010/postgres-api/internal/restaurants/dto"
	"github.com/alibaba0010/postgres-api/internal/restaurants/models"
	"github.com/alibaba0010/postgres-api/internal/utils"
	"go.uber.org/zap"
)

// CreateRestaurant creates a new restaurant
func CreateRestaurant(ctx context.Context, input dto.CreateRestaurantInput) (*dto.RestaurantResponse, *errors.AppError) {
	id, err := utils.GenerateUUIDv7()
	if err != nil {
		return nil, errors.InternalError(err)
	}

	restaurant := &models.Restaurant{
		ID:          id.String(),
		Name:        input.Name,
		Description: input.Description,
		Address:     input.Address,
		CuisineType: input.CuisineType,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	_, err = database.DB.NewInsert().Model(restaurant).Exec(ctx)
	if err != nil {
		logger.Log.Error("failed to create restaurant", zap.Error(err))
		return nil, errors.InternalError(err)
	}

	return &dto.RestaurantResponse{
		ID:          restaurant.ID,
		Name:        restaurant.Name,
		Description: restaurant.Description,
		Address:     restaurant.Address,
		CuisineType: restaurant.CuisineType,
		Rating:      restaurant.Rating,
		CreatedAt:   restaurant.CreatedAt.Format(time.RFC3339),
		UpdatedAt:   restaurant.UpdatedAt.Format(time.RFC3339),
	}, nil
}

// GetRestaurantByID retrieves a restaurant by ID
func GetRestaurantByID(ctx context.Context, id string) (*dto.RestaurantResponse, *errors.AppError) {
	restaurant := &models.Restaurant{}
	err := database.DB.NewSelect().Model(restaurant).Where("id = ?", id).Scan(ctx)
	if err != nil {
		return nil, errors.NotFoundError("restaurant not found")
	}

	return &dto.RestaurantResponse{
		ID:          restaurant.ID,
		Name:        restaurant.Name,
		Description: restaurant.Description,
		Address:     restaurant.Address,
		CuisineType: restaurant.CuisineType,
		Rating:      restaurant.Rating,
		CreatedAt:   restaurant.CreatedAt.Format(time.RFC3339),
		UpdatedAt:   restaurant.UpdatedAt.Format(time.RFC3339),
	}, nil
}

// GetAllRestaurants retrieves a paginated list of restaurants
func GetAllRestaurants(ctx context.Context, page, pageSize int, qStr string) ([]dto.RestaurantResponse, int64, *errors.AppError) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 20
	}

	restaurants := make([]models.Restaurant, 0)
	query := database.DB.NewSelect().Model(&restaurants)

	if qStr != "" {
		like := "%" + qStr + "%"
		query = query.Where("name ILIKE ? OR cuisine_type ILIKE ?", like, like)
	}

	// Count total
	countQuery := database.DB.NewSelect().Model((*models.Restaurant)(nil))
	if qStr != "" {
		like := "%" + qStr + "%"
		countQuery = countQuery.Where("name ILIKE ? OR cuisine_type ILIKE ?", like, like)
	}
	total, err := countQuery.Count(ctx)
	if err != nil {
		return nil, 0, errors.InternalError(err)
	}

	// Paginate
	query = query.Limit(pageSize).Offset((page - 1) * pageSize).Order("created_at DESC")

	if err := query.Scan(ctx); err != nil {
		return nil, 0, errors.InternalError(err)
	}

	// Map to DTO
	responses := make([]dto.RestaurantResponse, len(restaurants))
	for i, r := range restaurants {
		responses[i] = dto.RestaurantResponse{
			ID:          r.ID,
			Name:        r.Name,
			Description: r.Description,
			Address:     r.Address,
			CuisineType: r.CuisineType,
			Rating:      r.Rating,
			CreatedAt:   r.CreatedAt.Format(time.RFC3339),
			UpdatedAt:   r.UpdatedAt.Format(time.RFC3339),
		}
	}

	return responses, int64(total), nil
}

// UpdateRestaurant updates an existing restaurant
func UpdateRestaurant(ctx context.Context, id string, input dto.UpdateRestaurantInput) (*dto.RestaurantResponse, *errors.AppError) {
	restaurant := &models.Restaurant{}
	err := database.DB.NewSelect().Model(restaurant).Where("id = ?", id).Scan(ctx)
	if err != nil {
		return nil, errors.NotFoundError("restaurant not found")
	}

	if input.Name != "" {
		restaurant.Name = input.Name
	}
	if input.Description != "" {
		restaurant.Description = input.Description
	}
	if input.Address != "" {
		restaurant.Address = input.Address
	}
	if input.CuisineType != "" {
		restaurant.CuisineType = input.CuisineType
	}
	if input.Rating != nil {
		restaurant.Rating = *input.Rating
	}
	restaurant.UpdatedAt = time.Now()

	_, err = database.DB.NewUpdate().Model(restaurant).Where("id = ?", id).Exec(ctx)
	if err != nil {
		return nil, errors.InternalError(err)
	}

	return &dto.RestaurantResponse{
		ID:          restaurant.ID,
		Name:        restaurant.Name,
		Description: restaurant.Description,
		Address:     restaurant.Address,
		CuisineType: restaurant.CuisineType,
		Rating:      restaurant.Rating,
		CreatedAt:   restaurant.CreatedAt.Format(time.RFC3339),
		UpdatedAt:   restaurant.UpdatedAt.Format(time.RFC3339),
	}, nil
}
