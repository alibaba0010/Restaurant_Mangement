package services

import (
	"context"
	"time"

	"github.com/alibaba0010/postgres-api/internal/common/errors"
	"github.com/alibaba0010/postgres-api/internal/common/logger"
	"github.com/alibaba0010/postgres-api/internal/restaurants/dto"
	"github.com/alibaba0010/postgres-api/internal/restaurants/models"
	"github.com/alibaba0010/postgres-api/internal/restaurants/repositories"
	"github.com/alibaba0010/postgres-api/internal/utils"
	"go.uber.org/zap"
)
// MapRestaurantToResponse maps Restaurant model to RestaurantResponse DTO
func MapRestaurantToResponse(r *models.Restaurant) *dto.RestaurantResponse {
	var userIDStr *string
	if r.UserID != nil {
		idStr := r.UserID.String()
		userIDStr = &idStr
	}

	return &dto.RestaurantResponse{
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

	err = repositories.RestaurantRepo.Create(ctx, restaurant)
	if err != nil {
		logger.Log.Error("failed to create restaurant", zap.Error(err))
		return nil, errors.InternalError(err)
	}

	return MapRestaurantToResponse(restaurant), nil
}

// GetRestaurantByID retrieves a restaurant by ID
func GetRestaurantByID(ctx context.Context, id string) (*dto.RestaurantResponse, *errors.AppError) {
	restaurant, err := repositories.RestaurantRepo.FindByID(ctx, id)
	if err != nil {
		return nil, errors.NotFoundError("restaurant not found")
	}

	return MapRestaurantToResponse(restaurant), nil
}

// GetAllRestaurants retrieves a paginated list of restaurants
func GetAllRestaurants(ctx context.Context, page, pageSize int, qStr string, userID *string, sortBy, order string) ([]dto.RestaurantResponse, int64, *errors.AppError) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 20
	}

	restaurants, total, err := repositories.RestaurantRepo.FindAll(ctx, page, pageSize, qStr, userID, sortBy, order)
	if err != nil {
		return nil, 0, errors.InternalError(err)
	}

	// Map to DTO
	responses := make([]dto.RestaurantResponse, len(restaurants))
	for i, r := range restaurants {
		responses[i] = *MapRestaurantToResponse(&r)
	}

	return responses, total, nil
}

// UpdateRestaurant updates an existing restaurant
func UpdateRestaurant(ctx context.Context, id string, input dto.UpdateRestaurantInput) (*dto.RestaurantResponse, *errors.AppError) {
	restaurant, err := repositories.RestaurantRepo.FindByID(ctx, id)
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

	err = repositories.RestaurantRepo.Update(ctx, restaurant)
	if err != nil {
		return nil, errors.InternalError(err)
	}

	return MapRestaurantToResponse(restaurant), nil
}

