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
		ID:          id,
		Name:        input.Name,
		Description: input.Description,
		Address:     input.Address,
		AvatarURL:   input.AvatarURL,
		Status:      models.RestaurantStatusActive,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	// Set optional fields
	if input.Status != "" {
		restaurant.Status = models.RestaurantStatus(input.Status)
	}
	if input.UserID != nil {
		uuid, err := utils.ParseUUID(*input.UserID)
		if err == nil {
			restaurant.UserID = &uuid
		}
	}
	if input.Capacity != nil {
		restaurant.Capacity = *input.Capacity
	}
	if input.DeliveryAvailable != nil {
		restaurant.DeliveryAvailable = *input.DeliveryAvailable
	}
	if input.TakeawayAvailable != nil {
		restaurant.TakeawayAvailable = *input.TakeawayAvailable
	}

	_, err = database.DB.NewInsert().Model(restaurant).Exec(ctx)
	if err != nil {
		logger.Log.Error("failed to create restaurant", zap.Error(err))
		return nil, errors.InternalError(err)
	}

	var userIDStr *string
	if restaurant.UserID != nil {
		idStr := restaurant.UserID.String()
		userIDStr = &idStr
	}

	return &dto.RestaurantResponse{
		ID:                restaurant.ID.String(),
		Name:              restaurant.Name,
		Description:       restaurant.Description,
		Address:           restaurant.Address,
		AvatarURL:         restaurant.AvatarURL,
		Status:            string(restaurant.Status),
		UserID:            userIDStr,
		Capacity:          restaurant.Capacity,
		DeliveryAvailable: restaurant.DeliveryAvailable,
		TakeawayAvailable: restaurant.TakeawayAvailable,
		Rating:            restaurant.Rating,
		CreatedAt:         restaurant.CreatedAt.Format(time.RFC3339),
		UpdatedAt:         restaurant.UpdatedAt.Format(time.RFC3339),
	}, nil
}

// GetRestaurantByID retrieves a restaurant by ID
func GetRestaurantByID(ctx context.Context, id string) (*dto.RestaurantResponse, *errors.AppError) {
	restaurant := &models.Restaurant{}
	err := database.DB.NewSelect().Model(restaurant).Where("id = ?", id).Scan(ctx)
	if err != nil {
		return nil, errors.NotFoundError("restaurant not found")
	}

	var userIDStr *string
	if restaurant.UserID != nil {
		idStr := restaurant.UserID.String()
		userIDStr = &idStr
	}

	return &dto.RestaurantResponse{
		ID:                restaurant.ID.String(),
		Name:              restaurant.Name,
		Description:       restaurant.Description,
		Address:           restaurant.Address,
		AvatarURL:         restaurant.AvatarURL,
		Status:            string(restaurant.Status),
		UserID:            userIDStr,
		Capacity:          restaurant.Capacity,
		DeliveryAvailable: restaurant.DeliveryAvailable,
		TakeawayAvailable: restaurant.TakeawayAvailable,
		Rating:            restaurant.Rating,
		CreatedAt:         restaurant.CreatedAt.Format(time.RFC3339),
		UpdatedAt:         restaurant.UpdatedAt.Format(time.RFC3339),
	}, nil
}

// GetAllRestaurants retrieves a paginated list of restaurants
func GetAllRestaurants(ctx context.Context, page, pageSize int, qStr string, userID *string) ([]dto.RestaurantResponse, int64, *errors.AppError) {
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
		query = query.Where("name ILIKE ? OR description ILIKE ?", like, like)
	}

	if userID != nil {
		query = query.Where("user_id = ?", *userID)
	}

	// Count total
	countQuery := database.DB.NewSelect().Model((*models.Restaurant)(nil))
	if qStr != "" {
		like := "%" + qStr + "%"
		countQuery = countQuery.Where("name ILIKE ? OR description ILIKE ?", like, like)
	}
	if userID != nil {
		countQuery = countQuery.Where("user_id = ?", *userID)
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
		var userIDStr *string
		if r.UserID != nil {
			idStr := r.UserID.String()
			userIDStr = &idStr
		}

		responses[i] = dto.RestaurantResponse{
			ID:                r.ID.String(),
			Name:              r.Name,
			Description:       r.Description,
			Address:           r.Address,
			AvatarURL:         r.AvatarURL,
			Status:            string(r.Status),
			UserID:            userIDStr,
			Capacity:          r.Capacity,
			DeliveryAvailable: r.DeliveryAvailable,
			TakeawayAvailable: r.TakeawayAvailable,
			Rating:            r.Rating,
			CreatedAt:         r.CreatedAt.Format(time.RFC3339),
			UpdatedAt:         r.UpdatedAt.Format(time.RFC3339),
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
	if input.AvatarURL != "" {
		restaurant.AvatarURL = input.AvatarURL
	}
	if input.Status != "" {
		restaurant.Status = models.RestaurantStatus(input.Status)
	}
	if input.UserID != nil {
		uuid, err := utils.ParseUUID(*input.UserID)
		if err == nil {
			restaurant.UserID = &uuid
		}
	}
	if input.Capacity != nil {
		restaurant.Capacity = *input.Capacity
	}
	if input.DeliveryAvailable != nil {
		restaurant.DeliveryAvailable = *input.DeliveryAvailable
	}
	if input.TakeawayAvailable != nil {
		restaurant.TakeawayAvailable = *input.TakeawayAvailable
	}
	if input.Rating != nil {
		restaurant.Rating = *input.Rating
	}
	restaurant.UpdatedAt = time.Now()

	_, err = database.DB.NewUpdate().Model(restaurant).Where("id = ?", id).Exec(ctx)
	if err != nil {
		return nil, errors.InternalError(err)
	}

	var userIDStr *string
	if restaurant.UserID != nil {
		idStr := restaurant.UserID.String()
		userIDStr = &idStr
	}

	return &dto.RestaurantResponse{
		ID:                restaurant.ID.String(),
		Name:              restaurant.Name,
		Description:       restaurant.Description,
		Address:           restaurant.Address,
		AvatarURL:         restaurant.AvatarURL,
		Status:            string(restaurant.Status),
		UserID:            userIDStr,
		Capacity:          restaurant.Capacity,
		DeliveryAvailable: restaurant.DeliveryAvailable,
		TakeawayAvailable: restaurant.TakeawayAvailable,
		Rating:            restaurant.Rating,
		CreatedAt:         restaurant.CreatedAt.Format(time.RFC3339),
		UpdatedAt:         restaurant.UpdatedAt.Format(time.RFC3339),
	}, nil
}
