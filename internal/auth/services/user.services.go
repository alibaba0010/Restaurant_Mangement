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
	"github.com/google/uuid"
	"go.uber.org/zap"
)

func mapToCurrentUserResponse(user *models.User) dto.UserData {
	var primaryAddr string
	if user.AddressID != nil {
		for _, addr := range user.Addresses {
			if addr.ID == *user.AddressID {
				primaryAddr = addr.FormattedAddress
				break
			}
		}
	}

	return dto.UserData{
		ID:          user.ID,
		Name:        user.Name,
		Email:       user.Email,
		Role:        user.Role,
		AddressID:   utils.GetStringFromUUID(user.AddressID),
		Address:     primaryAddr,
		Addresses:   user.Addresses,
		Status:      user.Status,
		AvatarURL:   user.AvatarURL,
		PhoneNumber: user.PhoneNumber,
		CreatedAt:   user.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:   user.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}

type UserService interface {
	getUserByID(ctx context.Context, userID string) (*models.User, *errors.AppError)
	GetUserByID(ctx context.Context, userID string) (*dto.UserData, *errors.AppError)
	UpdateUser(ctx context.Context, userID string, input dto.UpdateUserInput) (*dto.UpdateUserResponse, *errors.AppError)
	GetAllUsers(ctx context.Context, page, pageSize int, qStr, role, sortBy, order string) ([]dto.UserData, int64, *errors.AppError)
	UpdateUserRoleStatus(ctx context.Context, userID string, input dto.UpdateUserRoleInput) (*dto.UpdateUserResponse, *errors.AppError)
	GetUserByEmail(ctx context.Context, email string) (*models.User, *errors.AppError)
	ValidateUserRole(roleStr string) (types.UserRole, *errors.AppError)
}

// getUserByID is the internal lightweight fetch (no addresses) — used by auth paths.
func getUserByID(ctx context.Context, userID string) (*models.User, *errors.AppError) {
	user, err := repositories.UserRepo.FindByID(ctx, userID)
	if err != nil {
		logger.Log.Debug("user not found by id", zap.String("user_id", userID))
		return nil, errors.NotFoundError("user not found")
	}
	return user, nil
}

// getUserByIDWithAddresses loads the user and their addresses — used for profile responses.
func getUserByIDWithAddresses(ctx context.Context, userID string) (*models.User, *errors.AppError) {
	user, err := repositories.UserRepo.FindByIDWithAddresses(ctx, userID)
	if err != nil {
		logger.Log.Debug("user not found by id", zap.String("user_id", userID))
		return nil, errors.NotFoundError("user not found")
	}
	return user, nil
}

func GetUserByID(ctx context.Context, userID string) (*dto.UserData, *errors.AppError) {
	user, appErr := getUserByIDWithAddresses(ctx, userID)
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

	// Fetch current user with addresses for response
	user, appErr := getUserByIDWithAddresses(ctx, userID)
	if appErr != nil {
		return nil, errors.NotFoundError("user not found")
	}

	// Track whether we made any DB changes
	didChange := false
	var fieldsToUpdate []string

	// Start a transaction for all changes
	tx, err := database.DB.BeginTx(ctx, nil)
	if err != nil {
		logger.Log.Error("failed to begin transaction", zap.Error(err))
		return nil, errors.TransactionError("starting", err)
	}

	// Defer rollback — no-op if already committed
	defer func() {
		if err := tx.Rollback(); err != nil && !strings.Contains(err.Error(), "already committed or rolled back") {
			logger.Log.Error("failed to rollback transaction", zap.Error(err))
		}
	}()

	// ── 1. Address update ────────────────────────────────────────────────────
	if input.Address != nil {
		addressSvc := address.NewService()
		fmtAddr, lat, lng, err := addressSvc.ProcessAddress(ctx, input.Address)
		if err != nil {
			return nil, errors.ToAppError(err)
		}

		// Parse userID as UUID for the FK
		parsedUID, err := uuid.Parse(userID)
		if err != nil {
			return nil, errors.ValidationError("invalid user ID")
		}

		rawAddr := input.Address.Address + ", " + input.Address.City + ", " + input.Address.Country
		addrModel := &address.AddressModel{
			UserID:           &parsedUID,
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
			// Unset existing defaults for this user before making this one the default
			_, _ = tx.NewUpdate().
				Model((*address.AddressModel)(nil)).
				Set("is_default = false").
				Where("user_id = ?", parsedUID).
				Exec(ctx)

			addrModel.ID = parsedAddrID
			addrModel.CreatedAt = time.Time{} // Don't overwrite created_at

			// Verify ownership before update
			exists, err := tx.NewSelect().
				Model((*address.AddressModel)(nil)).
				Where("id = ? AND user_id = ?", parsedAddrID, parsedUID).
				Exists(ctx)
			if err != nil || !exists {
				return nil, errors.ForbiddenError("Address not found or does not belong to user")
			}

			_, err = tx.NewUpdate().
				Model(addrModel).
				WherePK().
				Exec(ctx)
			if err != nil {
				logger.Log.Error("failed to update user address", zap.Error(err))
				return nil, errors.TransactionError("updating user address", err)
			}
		} else {
			// Create new address
			addrModel.ID = uuid.New()
			addrModel.CreatedAt = time.Now()

			// Unset existing defaults for this user before inserting new one
			_, _ = tx.NewUpdate().
				Model((*address.AddressModel)(nil)).
				Set("is_default = false").
				Where("user_id = ?", parsedUID).
				Exec(ctx)

			if _, err := tx.NewInsert().Model(addrModel).Exec(ctx); err != nil {
				logger.Log.Error("failed to insert user address", zap.Error(err))
				return nil, errors.TransactionError("adding user address", err)
			}
		}

		// Update User's primary AddressID
		user.AddressID = &addrModel.ID
		if !utils.Contains(fieldsToUpdate, "address_id") {
			fieldsToUpdate = append(fieldsToUpdate, "address_id")
		}
		
		user.Addresses = append(user.Addresses, addrModel)
		didChange = true
	}

	// ── 2. Phone number update ───────────────────────────────────────────────
	if input.PhoneNumber != "" && input.PhoneNumber != user.PhoneNumber {
		user.PhoneNumber = input.PhoneNumber
		if !utils.Contains(fieldsToUpdate, "phone_number") {
			fieldsToUpdate = append(fieldsToUpdate, "phone_number")
		}
		didChange = true
	}

	// ── 3. Persist user row changes ──────────────────────────────────────────
	if len(fieldsToUpdate) > 0 {
		user.UpdatedAt = time.Now()
		fieldsToUpdate = append(fieldsToUpdate, "updated_at")

		if err := repositories.UserRepo.Update(ctx, tx, user, fieldsToUpdate...); err != nil {
			logger.Log.Error("failed to update user info", zap.Error(err), zap.String("user_id", userID))
			return nil, errors.TransactionError("updating user profile", err)
		}
	}

	// Nothing changed at all — return current state without committing empty tx
	if !didChange {
		if err := tx.Rollback(); err != nil && !strings.Contains(err.Error(), "already committed or rolled back") {
			logger.Log.Warn("rollback on no-change", zap.Error(err))
		}
		return &dto.UpdateUserResponse{
			Title: "No changes",
			Data:  mapToCurrentUserResponse(user),
		}, nil
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		logger.Log.Error("failed to commit transaction", zap.Error(err))
		return nil, errors.TransactionError("committing user profile update", err)
	}

	return &dto.UpdateUserResponse{
		Title: "Updated successfully",
		Data:  mapToCurrentUserResponse(user),
	}, nil
}

// GetAllUsers returns a paginated, filtered and sorted list of users.
func GetAllUsers(ctx context.Context, page, pageSize int, qStr, role, sortBy, order string) ([]dto.UserData, int64, *errors.AppError) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 20
	}

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
	if err := utils.ValidateInput(input); err != nil {
		return nil, err
	}

	user, appErr := getUserByID(ctx, userID)
	if appErr != nil {
		return nil, errors.NotFoundError("user not found")
	}

	tx, err := database.DB.BeginTx(ctx, nil)
	if err != nil {
		return nil, errors.TransactionError("starting", err)
	}

	defer func() {
		if err := tx.Rollback(); err != nil && !strings.Contains(err.Error(), "already committed or rolled back") {
			logger.Log.Error("failed to rollback transaction", zap.Error(err))
		}
	}()

	var fieldsToUpdate []string

	if input.Role != "" && input.Role != user.Role {
		user.Role = input.Role
		fieldsToUpdate = append(fieldsToUpdate, "role")
	}

	if input.Status != "" && input.Status != user.Status {
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

	if err := repositories.UserRepo.Update(ctx, tx, user, fieldsToUpdate...); err != nil {
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
