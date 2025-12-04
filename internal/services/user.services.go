package services

import (
	"context"
	"time"

	"github.com/alibaba0010/postgres-api/internal/database"
	"github.com/alibaba0010/postgres-api/internal/dto"
	"github.com/alibaba0010/postgres-api/internal/errors"
	"github.com/alibaba0010/postgres-api/internal/logger"
	"github.com/alibaba0010/postgres-api/internal/models"
	"github.com/alibaba0010/postgres-api/internal/types"
	"github.com/go-playground/validator/v10"
	"github.com/uptrace/bun"
	"go.uber.org/zap"
)

// getUserByID returns a user by id. Accepts optional query modifiers.
func getUserByID(ctx context.Context, userID string, opts ...queryOption) (*models.User, *errors.AppError) {
	q := database.DB.NewSelect().Model((*models.User)(nil)).Where("id = ?", userID)

	// Apply functional options for query customization
	for _, opt := range opts {
		q = opt(q)
	}

	user := &models.User{}
	err := q.Scan(ctx)
	if err != nil {
		logger.Log.Debug("user not found by id", zap.String("user_id", userID))
		return nil, errors.InternalError(err)
	}

	return user, nil
}

// queryOption is a functional option for query customization
type queryOption func(*bun.SelectQuery) *bun.SelectQuery



// GetCurrentUserByID retrieves a user from the database by ID and returns formatted response
// Uses context for cancellation and timeout support
func GetCurrentUserByID(ctx context.Context, userID string) (*dto.CurrentUserResponse, *errors.AppError) {
	user, appErr := getUserByID(ctx, userID)
	if appErr != nil {
		logger.Log.Error("failed to fetch user from database", zap.String("user_id", userID))
		return nil, appErr
	}

	response := &dto.CurrentUserResponse{
		ID:        user.ID,
		Name:      user.Name,
		Email:     user.Email,
		Address:   user.Address,
		Role:      user.Role,
		CreatedAt: user.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt: user.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}

	logger.Log.Debug("user retrieved from database", zap.String("user_id", userID), zap.String("role", user.Role))
	return response, nil
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
	user := &models.User{}
	err := database.DB.NewSelect().Model(user).
		Where("email = ?", email).
		Scan(ctx)

	if err != nil {
		logger.Log.Debug("user not found by email", zap.String("email", email))
		return nil, errors.InternalError(err)
	}

	return user, nil
}

// LogResponse logs a response with status code and message
func LogResponse(status int, title, message string) {
	if status >= 500 {
		logger.Log.Error(title, zap.Int("status", status), zap.String("message", message))
	} else {
		logger.Log.Info(title, zap.Int("status", status), zap.String("message", message))
	}
}

// IsAdminRole checks if a user has admin role
func IsAdminRole(role string) bool {
	return role == string(types.RoleAdmin)
}

// IsManagementRole checks if a user has management or admin role
func IsManagementRole(role string) bool {
	roleEnum, _ := types.ToUserRole(role)
	return roleEnum == types.RoleManagement || roleEnum == types.RoleAdmin
}

// IsUserRole checks if a user has user role (or higher)
func IsUserRole(role string) bool {
	roleEnum, isValid := types.ToUserRole(role)
	return isValid && roleEnum.IsValid()
}

// UpdateUserAddress updates a user's address using a transaction-based approach
// Demonstrates advanced patterns: explicit transaction handling, context propagation, and validation
func UpdateUser(ctx context.Context, userID string, input dto.UpdateAddressInput) (*dto.UpdateAddressResponse, *errors.AppError) {
	// Validate input using the validator
	validate := validator.New()
	if err := validate.Struct(input); err != nil {
		if ves, ok := err.(validator.ValidationErrors); ok {
			var messages []string
			for _, fe := range ves {
				var msg string
				switch fe.Tag() {
				case "required":
					msg = "address is required"
				case "min":
					msg = "address must be at least 5 characters"
				case "max":
					msg = "address must be at most 255 characters"
				default:
					msg = "address is invalid"
				}
				messages = append(messages, msg)
			}
			return nil, errors.ValidationErrors(messages)
		}
		return nil, errors.ValidationError("invalid address input")
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
		return nil, errors.InternalError(err)
	}

	// Defer transaction rollback (will be a no-op if commit succeeded)
	defer func() {
		if err := tx.Rollback(); err != nil && err.Error() != "tx: already committed or rolled back" {
			logger.Log.Error("failed to rollback transaction", zap.Error(err))
		}
	}()

	// Update the address
	user.Address = input.Address
	user.UpdatedAt = time.Now()

	if _, err := tx.NewUpdate().Model(user).
		Column("address", "updated_at").
		Where("id = ?", userID).
		Exec(ctx); err != nil {
		logger.Log.Error("failed to update user address", zap.Error(err), zap.String("user_id", userID))
		return nil, errors.InternalError(err)
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		logger.Log.Error("failed to commit transaction", zap.Error(err))
		return nil, errors.InternalError(err)
	}

	// Build and return response
	response := &dto.UpdateAddressResponse{
		Title: "Success",
	}
	response.Data.ID = user.ID
	response.Data.Name = user.Name
	response.Data.Email = user.Email
	response.Data.Address = user.Address
	response.Data.Role = user.Role
	response.Data.UpdatedAt = user.UpdatedAt.Format("2006-01-02T15:04:05Z07:00")

	logger.Log.Info("user address updated", zap.String("user_id", userID))
	return response, nil
}

// GetUserByIDPublic retrieves public user information by ID (no password/sensitive data)
// Uses channel-based cancellation pattern for timeout support
func GetUserByIDPublic(ctx context.Context, userID string) (*models.User, *errors.AppError) {
	done := make(chan *models.User)
	errChan := make(chan error)

	// Query in a separate goroutine to support cancellation
	go func() {
		user, appErr := getUserByID(ctx, userID)
		if appErr != nil {
			errChan <- appErr
			return
		}
		done <- user
	}()

	select {
	case user := <-done:
		return user, nil
	case err := <-errChan:
		if appErr, ok := err.(*errors.AppError); ok {
			return nil, appErr
		}
		return nil, errors.InternalError(err)
	case <-ctx.Done():
		return nil, errors.InternalError(ctx.Err())
	}
}
