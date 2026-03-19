package services

import (
	"context"
	"strings"
	"time"

	"github.com/alibaba0010/postgres-api/internal/common/address"
	commonDto "github.com/alibaba0010/postgres-api/internal/common/dto"
	"github.com/alibaba0010/postgres-api/internal/common/errors"
	"github.com/alibaba0010/postgres-api/internal/common/logger"
	"github.com/alibaba0010/postgres-api/internal/common/types"
	"github.com/alibaba0010/postgres-api/internal/restaurants/dto"
	"github.com/alibaba0010/postgres-api/internal/restaurants/models"
	"github.com/alibaba0010/postgres-api/internal/restaurants/repositories"
	"github.com/alibaba0010/postgres-api/internal/utils"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
	"go.uber.org/zap"
)

// RestaurantService provides business logic for restaurant operations
type RestaurantService struct {
	repo             *repositories.RestaurantRepository
	addressService   address.AddressService
	db               *bun.DB // For transactions
}

type RestaurantServiceInterface interface {
	CreateRestaurant(ctx context.Context, input dto.CreateRestaurantInput, user *commonDto.AuthenticatedUser) (*dto.RestaurantResponse, *errors.AppError)
	GetRestaurantByID(ctx context.Context, id string) (*dto.RestaurantResponse, *errors.AppError)
	GetAllRestaurants(ctx context.Context, limit int, cursor string, qStr string, userID *string, status *string, sortBy, order string) ([]dto.RestaurantResponse, string, bool, int64, *errors.AppError)
	UpdateRestaurant(ctx context.Context, id string, input dto.UpdateRestaurantInput) (*dto.RestaurantResponse, *errors.AppError)
	DeleteRestaurant(ctx context.Context, id string, user *commonDto.AuthenticatedUser) (*dto.RestaurantResponse, *errors.AppError)
}
// NewRestaurantService creates and returns a new restaurant service instance
func NewRestaurantService(repo *repositories.RestaurantRepository, addressSvc address.AddressService, db *bun.DB) *RestaurantService {
	if addressSvc == nil {
		addressSvc = address.NewService()
	}
	return &RestaurantService{
		repo:           repo,
		addressService: addressSvc,
		db:             db,
	}
}

// MapToResponse maps Restaurant model to RestaurantResponse DTO
func (s *RestaurantService) MapToResponse(r *models.Restaurant) *dto.RestaurantResponse {
	var userIDStr *string
	if r.UserID != nil {
		idStr := r.UserID.String()
		userIDStr = &idStr
	}

	return &dto.RestaurantResponse{
		ID:                r.ID.String(),
		Name:              r.Name,
		Description:       r.Description,
		AddressID:         utils.GetStringFromUUID(r.AddressID),
		Addresses:         r.Addresses,
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



// Create creates a new restaurant
// user parameter is required and should be passed from the authentication middleware
func (s *RestaurantService) CreateRestaurant(ctx context.Context, input dto.CreateRestaurantInput, user *commonDto.AuthenticatedUser) (*dto.RestaurantResponse, *errors.AppError) {
	// Validate input
	if err := utils.ValidateInput(input); err != nil {
		return nil, err
	}

	// Sanitize string inputs to prevent injection attacks
	input.Name = strings.TrimSpace(input.Name)
	input.Description = strings.TrimSpace(input.Description)
	input.AvatarURL = strings.TrimSpace(input.AvatarURL)

	id, err := utils.GenerateUUIDv7()
	if err != nil {
		return nil, errors.InternalError(err)
	}

	var addrModel *address.AddressModel

	// Process address through Format -> Geocode pipeline
	if input.Address != nil {
		fmtAddr, lat, lng, err := s.addressService.ProcessAddress(ctx, input.Address)
		if err != nil {
			return nil, errors.ToAppError(err)
		}
		rawAddr := input.Address.Address + ", " + input.Address.City + ", " + input.Address.Country
		addrModel = &address.AddressModel{
			RestaurantID:     &id,
			FormattedAddress: fmtAddr,
			RawAddress:       rawAddr,
			Latitude:         lat,
			Longitude:        lng,
			IsDefault:        true,
		}
	}

	userID, err := uuid.Parse(user.UserID)
	if err != nil {
		logger.Log.Error("failed to parse user ID", zap.Error(err))
		return nil, errors.InternalError(err)
	}

	restaurant := &models.Restaurant{
		ID:          id,
		Name:        input.Name,
		Description: input.Description,
		AvatarURL:   input.AvatarURL,
		Status:      types.RestaurantStatusActive,
		UserID:      &userID,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if input.Status != "" {
		restaurant.Status = types.RestaurantStatus(input.Status)
	}
	if input.Capacity != nil && *input.Capacity >= 0 {
		restaurant.Capacity = *input.Capacity
	}
	if input.DeliveryAvailable != nil {
		restaurant.DeliveryAvailable = *input.DeliveryAvailable
	}
	if input.TakeawayAvailable != nil {
		restaurant.TakeawayAvailable = *input.TakeawayAvailable
	}

	// Run restaurant + optional address in a single transaction
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, errors.TransactionError("starting", err)
	}
	defer func() {
		if err := tx.Rollback(); err != nil && err.Error() != "tx: already committed or rolled back" {
			logger.Log.Error("failed to rollback transaction", zap.Error(err))
		}
	}()

	if _, err = tx.NewInsert().Model(restaurant).Exec(ctx); err != nil {
		logger.Log.Error("failed to create restaurant", zap.Error(err))
		return nil, errors.InternalError(err)
	}

	if addrModel != nil {
		// unset any previous defaults (none yet, but for consistency)
		_, _ = tx.NewUpdate().
			Model((*address.AddressModel)(nil)).
			Set("is_default = false").
			Where("restaurant_id = ?", id).
			Exec(ctx)

			if _, err = tx.NewInsert().Model(addrModel).Exec(ctx); err != nil {
				logger.Log.Error("failed to create restaurant address", zap.Error(err))
				return nil, errors.TransactionError("inserting restaurant address", err)
			}
			restaurant.AddressID = &addrModel.ID
			restaurant.Addresses = append(restaurant.Addresses, addrModel)
		}

	if err = tx.Commit(); err != nil {
		return nil, errors.TransactionError("committing restaurant creation", err)
	}

	return s.MapToResponse(restaurant), nil
}

// GetByID retrieves a restaurant by ID
func (s *RestaurantService) GetRestaurantByID(ctx context.Context, id string) (*dto.RestaurantResponse, *errors.AppError) {
	restaurant, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, errors.NotFoundError("restaurant not found")
	}

	return s.MapToResponse(restaurant), nil
}

// GetAll retrieves a paginated list of restaurants
func (s *RestaurantService) GetAllRestaurants(ctx context.Context, limit int, cursor string, qStr string, userID *string, status *string, sortBy, order string) ([]dto.RestaurantResponse, string, bool, int64, *errors.AppError) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}

	restaurants, nextCursor, hasMore, total, err := s.repo.FindAll(ctx, limit, cursor, qStr, userID, status, sortBy, order)
	if err != nil {
		return nil, "", false, 0, errors.InternalError(err)
	}

	// Map to DTO
	responses := make([]dto.RestaurantResponse, len(restaurants))
	for i, r := range restaurants {
		responses[i] = *s.MapToResponse(&r)
	}

	return responses, nextCursor, hasMore, total, nil
}



// Update updates an existing restaurant
func (s *RestaurantService) UpdateRestaurant(ctx context.Context, id string, input dto.UpdateRestaurantInput) (*dto.RestaurantResponse, *errors.AppError) {
	// Validate input
	if err := utils.ValidateInput(input); err != nil {
		return nil, err
	}

	// Fetch current restaurant to validate existence
	restaurant, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, errors.NotFoundError("restaurant not found")
	}

	// Start transaction for data consistency
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		logger.Log.Error("failed to begin transaction", zap.Error(err))
		return nil, errors.TransactionError("starting", err)
	}

	// Defer transaction rollback (will be a no-op if commit succeeded)
	defer func() {
		if err := tx.Rollback(); err != nil && err.Error() != "tx: already committed or rolled back" {
			logger.Log.Error("failed to rollback transaction", zap.Error(err))
		}
	}()

	// Apply field updates and get list of changed fields
	fieldsToUpdate, appErr := s.applyRestaurantUpdates(ctx, tx, restaurant, input)
	if appErr != nil {
		return nil, appErr
	}

	// If no fields changed, return early
	if len(fieldsToUpdate) == 0 {
		logger.Log.Debug("no changes detected for restaurant", zap.String("restaurant_id", id))
		return s.MapToResponse(restaurant), nil
	}

	// Set updated_at and add it to fields to update
	restaurant.UpdatedAt = time.Now()
	fieldsToUpdate = append(fieldsToUpdate, "updated_at")

	// Execute update within transaction
	err = s.repo.Update(ctx, tx, restaurant, fieldsToUpdate...)
	if err != nil {
		logger.Log.Error("failed to update restaurant", zap.Error(err), zap.String("restaurant_id", id))
		return nil, errors.TransactionError("updating restaurant", err)
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		logger.Log.Error("failed to commit transaction", zap.Error(err))
		return nil, errors.TransactionError("committing restaurant update", err)
	}

	logger.Log.Info("restaurant updated successfully", zap.String("restaurant_id", id), zap.Strings("updated_fields", fieldsToUpdate))

	return s.MapToResponse(restaurant), nil
}

// DeleteRestaurant deletes a restaurant by ID
func (s *RestaurantService) DeleteRestaurant(ctx context.Context, id string, user *commonDto.AuthenticatedUser) (*dto.RestaurantResponse, *errors.AppError) {
	// Fetch restaurant first to check existence and ownership
	restaurant, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, errors.NotFoundError("restaurant not found")
	}

	// Authorization Check
	if user.Role != types.RoleAdmin {
		if user.Role == types.RoleManagement {
			if restaurant.UserID == nil || restaurant.UserID.String() != user.UserID {
				return nil, errors.ForbiddenError("You do not have permission to delete this restaurant")
			}
		} else {
			return nil, errors.ForbiddenError("You do not have permission to perform this action")
		}
	}

	// Start transaction
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		logger.Log.Error("failed to begin transaction", zap.Error(err))
		return nil, errors.TransactionError("starting", err)
	}

	// Defer rollback
	defer func() {
		if err := tx.Rollback(); err != nil && err.Error() != "tx: already committed or rolled back" {
			logger.Log.Error("failed to rollback transaction", zap.Error(err))
		}
	}()

	// Perform Delete
	if err := s.repo.Delete(ctx, tx, id); err != nil {
		logger.Log.Error("failed to delete restaurant", zap.Error(err), zap.String("restaurant_id", id))
		return nil, errors.InternalError(err)
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		logger.Log.Error("failed to commit transaction", zap.Error(err))
		return nil, errors.TransactionError("committing restaurant deletion", err)
	}

	logger.Log.Info("restaurant deleted successfully", zap.String("restaurant_id", id), zap.String("user_id", user.UserID))

	return s.MapToResponse(restaurant), nil
}
	

// applyRestaurantUpdates applies field updates from input to the restaurant model,
// using the provided transaction for any address inserts.
func (s *RestaurantService) applyRestaurantUpdates(ctx context.Context, tx bun.Tx, restaurant *models.Restaurant, input dto.UpdateRestaurantInput) ([]string, *errors.AppError) {
	var fieldsToUpdate []string

	// Update name if provided and different
	if input.Name != "" {
		sanitizedName := strings.TrimSpace(input.Name)
		if sanitizedName != restaurant.Name {
			restaurant.Name = sanitizedName
			fieldsToUpdate = append(fieldsToUpdate, "name")
		}
	}

	// Update description if provided and different
	if input.Description != "" {
		sanitizedDesc := strings.TrimSpace(input.Description)
		if sanitizedDesc != restaurant.Description {
			restaurant.Description = sanitizedDesc
			fieldsToUpdate = append(fieldsToUpdate, "description")
		}
	}

	// Update address — update existing if ID provided, or insert a new one
	if input.Address != nil {
		fmtAddr, lat, lng, err := s.addressService.ProcessAddress(ctx, input.Address)
		if err != nil {
			return nil, errors.ToAppError(err)
		}

		rawAddr := input.Address.Address + ", " + input.Address.City + ", " + input.Address.Country
		addrModel := &address.AddressModel{
			RestaurantID:     &restaurant.ID,
			FormattedAddress: fmtAddr,
			RawAddress:       rawAddr,
			Latitude:         lat,
			Longitude:        lng,
			IsDefault:        true,
			UpdatedAt:        time.Now(),
		}

		if input.Address.ID != "" {
			// Update existing address
			parsedAddrID, err := uuid.Parse(input.Address.ID)
			if err != nil {
				return nil, errors.ValidationError("invalid address ID")
			}
			addrModel.ID = parsedAddrID

			// Verify ownership
			exists, err := tx.NewSelect().
				Model((*address.AddressModel)(nil)).
				Where("id = ? AND restaurant_id = ?", parsedAddrID, restaurant.ID).
				Exists(ctx)
			if err != nil || !exists {
				return nil, errors.ForbiddenError("Address not found or does not belong to restaurant")
			}

			_, err = tx.NewUpdate().
				Model(addrModel).
				WherePK().
				Exec(ctx)
			if err != nil {
				logger.Log.Error("failed to update restaurant address", zap.Error(err))
				return nil, errors.TransactionError("updating restaurant address", err)
			}
		} else {
			// Create new address
			addrModel.ID = uuid.New()
			addrModel.CreatedAt = time.Now()

			// Unset previous defaults for this restaurant
			_, _ = tx.NewUpdate().
				Model((*address.AddressModel)(nil)).
				Set("is_default = false").
				Where("restaurant_id = ?", restaurant.ID).
				Exec(ctx)

			if _, err := tx.NewInsert().Model(addrModel).Exec(ctx); err != nil {
				logger.Log.Error("failed to insert restaurant address", zap.Error(err))
				return nil, errors.TransactionError("inserting restaurant address", err)
			}
		}

		// Update Restaurant's primary AddressID
		restaurant.AddressID = &addrModel.ID
		if !utils.Contains(fieldsToUpdate, "address_id") {
			fieldsToUpdate = append(fieldsToUpdate, "address_id")
		}

		restaurant.Addresses = append(restaurant.Addresses, addrModel)
	}

	// Update avatar URL if provided and different
	if input.AvatarURL != "" {
		sanitizedURL := strings.TrimSpace(input.AvatarURL)
		if sanitizedURL != restaurant.AvatarURL {
			restaurant.AvatarURL = sanitizedURL
			fieldsToUpdate = append(fieldsToUpdate, "avatar_url")
		}
	}

	// Update status if provided and different
	if input.Status != "" {
		newStatus := types.RestaurantStatus(input.Status)
		if newStatus != restaurant.Status {
			restaurant.Status = newStatus
			fieldsToUpdate = append(fieldsToUpdate, "status")
		}
	}

	// Update user ID if provided and different
	if input.UserID != nil {
		userID, err := utils.ParseUUID(*input.UserID)
		if err == nil {
			// Compare the parsed UUID with current UserID
			if restaurant.UserID == nil || restaurant.UserID.String() != userID.String() {
				restaurant.UserID = &userID
				fieldsToUpdate = append(fieldsToUpdate, "user_id")
			}
		}
	}

	// Update capacity if provided and different
	if input.Capacity != nil {
		if *input.Capacity != restaurant.Capacity {
			restaurant.Capacity = *input.Capacity
			fieldsToUpdate = append(fieldsToUpdate, "capacity")
		}
	}

	// Update delivery available if provided and different
	if input.DeliveryAvailable != nil {
		if *input.DeliveryAvailable != restaurant.DeliveryAvailable {
			restaurant.DeliveryAvailable = *input.DeliveryAvailable
			fieldsToUpdate = append(fieldsToUpdate, "delivery_available")
		}
	}

	// Update takeaway available if provided and different
	if input.TakeawayAvailable != nil {
		if *input.TakeawayAvailable != restaurant.TakeawayAvailable {
			restaurant.TakeawayAvailable = *input.TakeawayAvailable
			fieldsToUpdate = append(fieldsToUpdate, "takeaway_available")
		}
	}

	// Update rating if provided and different
	if input.Rating != nil {
		if *input.Rating != restaurant.Rating {
			restaurant.Rating = *input.Rating
			fieldsToUpdate = append(fieldsToUpdate, "rating")
		}
	}

	return fieldsToUpdate, nil
}