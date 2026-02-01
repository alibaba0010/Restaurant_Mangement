package services

import (
	"context"
	"strings"
	"time"

	"github.com/alibaba0010/postgres-api/internal/auth/dto"
	"github.com/alibaba0010/postgres-api/internal/auth/models"
	"github.com/alibaba0010/postgres-api/internal/auth/repositories"
	"github.com/alibaba0010/postgres-api/internal/common/address"
	"github.com/alibaba0010/postgres-api/internal/common/errors"
	"github.com/alibaba0010/postgres-api/internal/common/logger"
	"github.com/alibaba0010/postgres-api/internal/common/types"
	"github.com/alibaba0010/postgres-api/internal/database"
	"github.com/alibaba0010/postgres-api/internal/utils"
	"github.com/go-playground/validator/v10"
	"go.uber.org/zap"
)

// mapToCurrentUserResponse converts a user model to a user DTO
func mapToCurrentUserResponse(user *models.User) dto.UserData {
	return dto.UserData{
		ID:          user.ID,
		Name:        user.Name,
		Email:       user.Email,
		Address:     user.Address,
		Role:        user.Role,
		Status:      user.Status,
		AvatarURL:   user.AvatarURL,
		PhoneNumber: user.PhoneNumber,
		Latitude:    user.Latitude,
		Longitude:   user.Longitude,
		CreatedAt:   user.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:   user.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}

type UserService interface {
	// getUserByID returns a user by id using repository
	getUserByID(ctx context.Context, userID string) (*models.User, *errors.AppError)
	// GetUserByID retrieves user information by ID for either current user or by admin
	GetUserByID(ctx context.Context, userID string) (*dto.UserData, *errors.AppError)
	// UpdateUser updates a user's address or phone number
	UpdateUser(ctx context.Context, userID string, input dto.UpdateUserInput) (*dto.UpdateUserResponse, *errors.AppError)
	// GetAllUsers returns a paginated, filtered and sorted list of users.
	GetAllUsers(ctx context.Context, page, pageSize int, qStr, role, sortBy, order string) ([]dto.UserData, int64, *errors.AppError)
	// UpdateUserRoleStatus updates a user's role and/or status (admin or management)
	UpdateUserRoleStatus(ctx context.Context, userID string, input dto.UpdateUserRoleInput) (*dto.UpdateUserResponse, *errors.AppError)
	// GetUserByEmail retrieves a user by their email address
	GetUserByEmail(ctx context.Context, email string) (*models.User, *errors.AppError)
	// ValidateUserRole validates and converts a role string to UserRole type
	ValidateUserRole(roleStr string) (types.UserRole, *errors.AppError)
}

func getUserByID(ctx context.Context, userID string) (*models.User, *errors.AppError) {
	user, err := repositories.UserRepo.FindByID(ctx, userID)
	if err != nil {
		logger.Log.Debug("user not found by id", zap.String("user_id", userID))
		return nil, errors.NotFoundError("user not found")
	}
	return user, nil
}

func GetUserByID(ctx context.Context, userID string) (*dto.UserData, *errors.AppError) {
		user, appErr := getUserByID(ctx, userID)
	if appErr != nil {
		logger.Log.Error("failed to fetch user from database", zap.String("user_id", userID))
		return nil, appErr
	}

	response := mapToCurrentUserResponse(user)
	return &response, nil
}

func UpdateUser(ctx context.Context, userID string, input dto.UpdateUserInput) (*dto.UpdateUserResponse, *errors.AppError) {
	// Validate input using the validator
	if err := utils.ValidateInput(input); err != nil {
		return nil, err
	}

	// Fetch current user first to validate existence
	user, appErr := getUserByID(ctx, userID)
	if appErr != nil {
		return nil, errors.NotFoundError("user not found")
	}

	// Update address using explicit transaction for data consistency
	tx, err := database.DB.BeginTx(ctx, nil)
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

	// Update the fields if provided
	var fieldsToUpdate []string

	// Update address if provided using Format -> Geocode pipeline
	if input.Address != nil {
		addressSvc := address.NewService()
		fmtAddr, lat, lng, appErr := addressSvc.ProcessAddress(ctx, input.Address)
		if appErr != nil {
			return nil, appErr
		}

		// Only update if the formatted address has changed
		if fmtAddr != user.Address {
			user.Address = fmtAddr
			user.Latitude = lat
			user.Longitude = lng
			fieldsToUpdate = append(fieldsToUpdate, "address", "latitude", "longitude")
		}
	}

	// Update phone number if provided and different
	if input.PhoneNumber != "" && input.PhoneNumber != user.PhoneNumber {
		user.PhoneNumber = input.PhoneNumber
		fieldsToUpdate = append(fieldsToUpdate, "phone_number")
	}

	// If no fields changed, return early
	if len(fieldsToUpdate) == 0 {
		return &dto.UpdateUserResponse{
			Title: "No changes",
			Data:  mapToCurrentUserResponse(user),
		}, nil
	}

	// Set updated_at and add to fields to update
	user.UpdatedAt = time.Now()
	fieldsToUpdate = append(fieldsToUpdate, "updated_at")

	// Execute update within transaction
	err = repositories.UserRepo.Update(ctx, tx, user, fieldsToUpdate...)
	if err != nil {
		logger.Log.Error("failed to update user info", zap.Error(err), zap.String("user_id", userID))
		return nil, errors.TransactionError("updating user profile", err)
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		logger.Log.Error("failed to commit transaction", zap.Error(err))
		return nil, errors.TransactionError("committing user profile update", err)
	}

	// Build and return response
	response := &dto.UpdateUserResponse{
		Title: "Success",
		Data:  mapToCurrentUserResponse(user),
	}

	return response, nil
}

// GetAllUsers returns a paginated, filtered and sorted list of users.
// Supports search by name/email (`q`), role filter, and sorting by allowed columns.
func GetAllUsers(ctx context.Context, page, pageSize int, qStr, role, sortBy, order string) ([]dto.UserData, int64, *errors.AppError) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 20
	}

	// Sorting validation
	allowedSort := map[string]bool{"name": true, "email": true, "created_at": true, "role": true}
	if !allowedSort[sortBy] {
		sortBy = "created_at"
	}
	if strings.ToLower(order) != "asc" {
		order = "DESC"
	} else {
		order = "ASC"
	}

	users, total, err := repositories.UserRepo.FindAll(ctx, page, pageSize, qStr, role, sortBy, order)
	if err != nil {
		logger.Log.Error("failed to fetch users", zap.Error(err))
		return nil, 0, errors.NotFoundError("no users found")
	}

	// Map to DTO
	result := make([]dto.UserData, 0, len(users))
	for _, u := range users {
		result = append(result, mapToCurrentUserResponse(&u))
	}

	return result, total, nil
}

func ValidateUserRole(roleStr string) (types.UserRole, *errors.AppError) {
	role, isValid := types.ToUserRole(roleStr)
	if !isValid {
		logger.Log.Warn("invalid user role", zap.String("role", roleStr))
		return "", errors.ValidationError("invalid user role: " + roleStr)
	}
	return role, nil
}

func GetUserByEmail(ctx context.Context, email string) (*models.User, *errors.AppError) {
	user, err := repositories.UserRepo.FindByEmail(ctx, email)
	if err != nil {
		logger.Log.Debug("user not found by email", zap.String("email", email))
		return nil, errors.InternalError(err)
	}

	return user, nil
}

func UpdateUserRoleStatus(ctx context.Context, userID string, input dto.UpdateUserRoleInput) (*dto.UpdateUserResponse, *errors.AppError) {
	// Validate input
	validate := validator.New()
	if err := validate.Struct(input); err != nil {
		return nil, errors.ValidationError("invalid role: " + string(input.Role))
	}

	// Fetch user
	user, appErr := getUserByID(ctx, userID)
	if appErr != nil {
		return nil, errors.NotFoundError("user not found")
	}

	// Start transaction
	tx, err := database.DB.BeginTx(ctx, nil)
	if err != nil {
		return nil, errors.TransactionError("starting", err)
	}

	// Defer transaction rollback (will be a no-op if commit succeeded)
	defer func() {
		if err := tx.Rollback(); err != nil && err.Error() != "tx: already committed or rolled back" {
			logger.Log.Error("failed to rollback transaction", zap.Error(err))
		}
	}()

	// Update role/status - track which fields actually changed
	var fieldsToUpdate []string

	// Update role if provided and different
	if input.Role != "" && input.Role != user.Role {
		user.Role = input.Role
		fieldsToUpdate = append(fieldsToUpdate, "role")
	}

	// Update status if provided and different
	if input.Status != "" && input.Status != user.Status {
		user.Status = input.Status
		fieldsToUpdate = append(fieldsToUpdate, "status")
	}

	// If no fields changed, return early
	if len(fieldsToUpdate) == 0 {
		return &dto.UpdateUserResponse{
			Title: "No changes",
			Data:  mapToCurrentUserResponse(user),
		}, nil
	}

	// Set updated_at and add to fields to update
	user.UpdatedAt = time.Now()
	fieldsToUpdate = append(fieldsToUpdate, "updated_at")

	// Execute update within transaction
	err = repositories.UserRepo.Update(ctx, tx, user, fieldsToUpdate...)
	if err != nil {
		return nil, errors.TransactionError("updating user role and status", err)
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return nil, errors.TransactionError("committing user role/status update", err)
	}

	return &dto.UpdateUserResponse{
		Title: "Updated User Successfully",
		Data:  mapToCurrentUserResponse(user),
	}, nil
}
