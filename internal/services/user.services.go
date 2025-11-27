package services

import (
	"context"

	"github.com/alibaba0010/postgres-api/internal/database"
	"github.com/alibaba0010/postgres-api/internal/errors"
	"github.com/alibaba0010/postgres-api/internal/logger"
	"github.com/alibaba0010/postgres-api/internal/models"
	"github.com/alibaba0010/postgres-api/internal/types"
	"github.com/alibaba0010/postgres-api/internal/dto"
	"go.uber.org/zap"
)



// GetCurrentUserByID retrieves a user from the database by ID and returns formatted response
func GetCurrentUserByID(ctx context.Context, userID string) (*dto.CurrentUserResponse, *errors.AppError) {
	user := &models.User{}
	err := database.DB.NewSelect().Model(user).
		Where("id = ?", userID).
		Scan(ctx)

	if err != nil {
		logger.Log.Error("failed to fetch user from database", zap.Error(err), zap.String("user_id", userID))
		return nil, errors.InternalError(err)
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

// CheckRolePermission verifies if a user role has permission to access a required role
// Returns true if the user's role meets or exceeds the required role
func CheckRolePermission(userRole string, requiredRoles ...string) bool {
	userRoleEnum, isValid := types.ToUserRole(userRole)
	if !isValid {
		logger.Log.Warn("invalid user role in permission check", zap.String("role", userRole))
		return false
	}

	for _, requiredRole := range requiredRoles {
		requiredRoleEnum, isValid := types.ToUserRole(requiredRole)
		if !isValid {
			logger.Log.Warn("invalid required role", zap.String("role", requiredRole))
			continue
		}

		if userRoleEnum.HasPermission(requiredRoleEnum) {
			return true
		}
	}

	return false
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
