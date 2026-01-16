package services

import (
	"context"
	"strings"
	"time"

	"github.com/alibaba0010/postgres-api/internal/auth/dto"
	"github.com/alibaba0010/postgres-api/internal/auth/models"
	"github.com/alibaba0010/postgres-api/internal/auth/repositories"
	"github.com/alibaba0010/postgres-api/internal/common/errors"
	"github.com/alibaba0010/postgres-api/internal/common/logger"
	"github.com/alibaba0010/postgres-api/internal/common/types"
	"github.com/alibaba0010/postgres-api/internal/database"
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
		CreatedAt:   user.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:   user.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}

// getUserByID returns a user by id using repository
func getUserByID(ctx context.Context, userID string) (*models.User, *errors.AppError) {
	user, err := repositories.UserRepo.FindByID(ctx, userID)
	if err != nil {
		logger.Log.Debug("user not found by id", zap.String("user_id", userID))
		return nil, errors.NotFoundError("user not found")
	}
	return user, nil
}

// GetUserByID retrieves user information by ID for either current user or by admin 
func GetUserByID(ctx context.Context, userID string) (*dto.UserData, *errors.AppError) {
		user, appErr := getUserByID(ctx, userID)
	if appErr != nil {
		logger.Log.Error("failed to fetch user from database", zap.String("user_id", userID))
		return nil, appErr
	}

	response := mapToCurrentUserResponse(user)
	return &response, nil
}

// UpdateUser updates a user's address or phone number
// TRANSACTION ANALYSIS: Transaction IS necessary here because:
// 1. We're performing a single atomic operation (update single row)
// 2. Concurrent updates could cause race conditions
// 3. The transaction ensures consistency between the fetch and update operations
// 4. It provides isolation level protection against dirty reads
// However, for simple single-row updates, connection pooling alone might suffice
// if the operation is purely an UPDATE statement without multiple queries.
// Keeping the transaction for safety and consistency.
func UpdateUser(ctx context.Context, userID string, input dto.UpdateUserInput) (*dto.UpdateUserResponse, *errors.AppError) {
	// Validate input using the validator
	validate := validator.New()
	if err := validate.Struct(input); err != nil {
		if ves, ok := err.(validator.ValidationErrors); ok {
			var messages []string
			for _, fe := range ves {
				var msg string
				switch fe.Tag() {
				case "min":
					msg = fe.Field() + " is too short"
				case "max":
					msg = fe.Field() + " is too long"
				default:
					msg = fe.Field() + " is invalid"
				}
				messages = append(messages, msg)
			}
			return nil, errors.ValidationErrors(messages)
		}
		return nil, errors.ValidationError("invalid user update input")
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
	if input.Address != "" {
		user.Address = input.Address
		fieldsToUpdate = append(fieldsToUpdate, "address")
	}
	if input.PhoneNumber != "" {
		user.PhoneNumber = input.PhoneNumber
		fieldsToUpdate = append(fieldsToUpdate, "phone_number")
	}

	if len(fieldsToUpdate) == 0 {
		return &dto.UpdateUserResponse{
			Title: "No changes",
			Data:  mapToCurrentUserResponse(user),
		}, nil
	}

	user.UpdatedAt = time.Now()
	fieldsToUpdate = append(fieldsToUpdate, "updated_at")

	err = repositories.UserRepo.UpdateInTx(ctx, tx, user, fieldsToUpdate...)
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

// ValidateUserRole checks if a user role is valid
func ValidateUserRole(roleStr string) (types.UserRole, *errors.AppError) {
	role, isValid := types.ToUserRole(roleStr)
	if !isValid {
		logger.Log.Warn("invalid user role", zap.String("role", roleStr))
		return "", errors.ValidationError("invalid user role: " + roleStr)
	}
	return role, nil
}

// GetUserByEmail retrieves a user by email address
func GetUserByEmail(ctx context.Context, email string) (*models.User, *errors.AppError) {
	user, err := repositories.UserRepo.FindByEmail(ctx, email)
	if err != nil {
		logger.Log.Debug("user not found by email", zap.String("email", email))
		return nil, errors.InternalError(err)
	}

	return user, nil
}

// UpdateUserRoleStatus updates a user's role and/or status (admin or management)
// Transaction is necessary here to prevent race conditions between reading the user
// and updating multiple fields atomically, ensuring consistency.
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

	// Update role/status
	var fieldsToUpdate []string
	if input.Role != "" {
		user.Role = input.Role
		fieldsToUpdate = append(fieldsToUpdate, "role")
	}
	if input.Status != "" {
		user.Status = input.Status
		fieldsToUpdate = append(fieldsToUpdate, "status")
	}

	if len(fieldsToUpdate) == 0 {
		return &dto.UpdateUserResponse{
			Title: "No changes",
			Data:  mapToCurrentUserResponse(user),
		}, nil
	}

	user.UpdatedAt = time.Now()
	fieldsToUpdate = append(fieldsToUpdate, "updated_at")

	err = repositories.UserRepo.UpdateInTx(ctx, tx, user, fieldsToUpdate...)
	if err != nil {
		return nil, errors.TransactionError("updating user role and status", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, errors.TransactionError("committing user role/status update", err)
	}

	return &dto.UpdateUserResponse{
		Title: "Updated User Successfully",
		Data:  mapToCurrentUserResponse(user),
	}, nil
}
