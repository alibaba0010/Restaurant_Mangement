package services

import (
	"context"
	"time"

	commondto "github.com/alibaba0010/postgres-api/internal/common/dto"
	"github.com/alibaba0010/postgres-api/internal/common/errors"
	"github.com/alibaba0010/postgres-api/internal/common/events"
	"github.com/alibaba0010/postgres-api/internal/common/guards"
	"github.com/alibaba0010/postgres-api/internal/common/logger"
	commontypes "github.com/alibaba0010/postgres-api/internal/common/types"
	"github.com/alibaba0010/postgres-api/internal/restaurants/dto"
	"github.com/alibaba0010/postgres-api/internal/restaurants/models"
	"github.com/alibaba0010/postgres-api/internal/restaurants/repositories"
	"github.com/alibaba0010/postgres-api/internal/utils"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// CategoryService provides business logic for menu category operations,
// including authorization and input validation.
type CategoryService struct {
	repo           *repositories.CategoryRepository
	restaurantRepo *repositories.RestaurantRepository
}

// CategoryServiceInterface defines the interface for category business operations.
type CategoryServiceInterface interface {
	// CreateCategory validates input and user permissions before creating a category
	CreateCategory(ctx context.Context, user *commondto.AuthenticatedUser, input dto.CreateCategoryInput) (*dto.CategoryResponse, *errors.AppError)
	// ListAllCategories retrieves all categories available in the system
	ListAllCategories(ctx context.Context) ([]dto.CategoryResponse, *errors.AppError)
	// UpdateCategory updates an existing category
	UpdateCategory(ctx context.Context, user *commondto.AuthenticatedUser, id string, input dto.UpdateCategoryInput) (*dto.CategoryResponse, *errors.AppError)
	// DeleteCategory removes a category
	DeleteCategory(ctx context.Context, user *commondto.AuthenticatedUser, id string) *errors.AppError
}

// NewCategoryService creates a new instance of CategoryService.
func NewCategoryService(repo *repositories.CategoryRepository, restaurantRepo *repositories.RestaurantRepository) *CategoryService {
	return &CategoryService{
		repo:           repo,
		restaurantRepo: restaurantRepo,
	}
}

// MapCategoryToResponse transforms a MenuCategory model into a CategoryResponse DTO.
// This decouples the internal database structure from the public API contract.
func (s *CategoryService) MapCategoryToResponse(c *models.MenuCategory) dto.CategoryResponse {
	return dto.CategoryResponse{
		ID:           c.ID.String(),
		RestaurantID: c.RestaurantID.String(),
		Name:         c.Name,
		Description:  c.Description,
		SortOrder:    c.SortOrder,
		CreatedAt:    c.CreatedAt.Format(time.RFC3339),
		UpdatedAt:    c.UpdatedAt.Format(time.RFC3339),
	}
}

// CreateCategory handles the full workflow of adding a new category to a restaurant.
// It performs:
// 1. Ownership authorization (only restaurant owners or admins can add categories).
// 2. Structural validation of the input DTO.
// 3. UUID parsing and model preparation.
// 4. Persistence via the repository.
func (s *CategoryService) CreateCategory(ctx context.Context, user *commondto.AuthenticatedUser, input dto.CreateCategoryInput) (*dto.CategoryResponse, *errors.AppError) {
	// Step 1: Authorization check.
	// Users with management role must prove they own the restaurant they are modifying.
	if user.Role == commontypes.RoleManagement {
		if appErr := guards.AuthorizeRestaurantOwner(ctx, s.restaurantRepo, input.RestaurantID, user.UserID); appErr != nil {
			return nil, appErr
		}
	}

	// Step 2: Validate input fields (e.g., Name cannot be empty).
	if err := utils.ValidateInput(input); err != nil {
		return nil, err
	}

	// Step 3: Parse IDs and initialize model.
	rID, err := uuid.Parse(input.RestaurantID)
	if err != nil {
		return nil, errors.ValidationError("Invalid restaurant ID")
	}

	category := &models.MenuCategory{
		RestaurantID: rID,
		Name:         input.Name,
		Description:  input.Description,
		SortOrder:    input.SortOrder,
	}

	// Step 4: Persist to database.
	err = s.repo.Create(ctx, category)
	if err != nil {
		return nil, errors.InternalError(err)
	}

	// Return mapped response.
	resp := s.MapCategoryToResponse(category)

	// Invalidate Menu Cache.
	utils.InvalidateCacheByPrefix(ctx, "menus:list:")

	// Publish Event.
	s.publishCategoryEvent(ctx, category.ID.String(), "category.created")

	return &resp, nil
}

// ListCategoriesByRestaurant returns all categories associated with a given restaurant.
// It is used to populate category filters or menu navigation in the frontend.
func (s *CategoryService) ListCategoriesByRestaurant(ctx context.Context, restaurantID string) ([]dto.CategoryResponse, *errors.AppError) {
	categories, err := s.repo.FindByRestaurantID(ctx, restaurantID)
	if err != nil {
		return nil, errors.InternalError(err)
	}

	// Map models to DTOs for the response.
	responses := make([]dto.CategoryResponse, len(categories))
	for i, c := range categories {
		responses[i] = s.MapCategoryToResponse(&c)
	}

	return responses, nil
}

// ListAllCategories returns all categories available in the database.
func (s *CategoryService) ListAllCategories(ctx context.Context) ([]dto.CategoryResponse, *errors.AppError) {
	categories, err := s.repo.ListAll(ctx)
	if err != nil {
		return nil, errors.InternalError(err)
	}

	responses := make([]dto.CategoryResponse, len(categories))
	for i, c := range categories {
		responses[i] = s.MapCategoryToResponse(&c)
	}

	return responses, nil
}

// UpdateCategory updates an existing category's details.
func (s *CategoryService) UpdateCategory(ctx context.Context, user *commondto.AuthenticatedUser, id string, input dto.UpdateCategoryInput) (*dto.CategoryResponse, *errors.AppError) {
	category, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, errors.NotFoundError("Category not found")
	}

	// Authorization check.
	if user.Role == commontypes.RoleManagement {
		if appErr := guards.AuthorizeRestaurantOwner(ctx, s.restaurantRepo, category.RestaurantID.String(), user.UserID); appErr != nil {
			return nil, appErr
		}
	}

	if input.Name != nil {
		category.Name = *input.Name
	}
	if input.Description != nil {
		category.Description = *input.Description
	}
	if input.SortOrder != nil {
		category.SortOrder = *input.SortOrder
	}

	err = s.repo.Update(ctx, nil, category)
	if err != nil {
		return nil, errors.InternalError(err)
	}

	// Invalidate Menu Cache because menus linked to this category might need update.
	utils.InvalidateCacheByPrefix(ctx, "menus:list:")

	// Publish Event.
	s.publishCategoryEvent(ctx, id, "category.updated")

	return &dto.CategoryResponse{
		ID:           category.ID.String(),
		RestaurantID: category.RestaurantID.String(),
		Name:         category.Name,
		Description:  category.Description,
		SortOrder:    category.SortOrder,
		CreatedAt:    category.CreatedAt.Format(time.RFC3339),
		UpdatedAt:    category.UpdatedAt.Format(time.RFC3339),
	}, nil
}

// DeleteCategory removes a category.
func (s *CategoryService) DeleteCategory(ctx context.Context, user *commondto.AuthenticatedUser, id string) *errors.AppError {
	category, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return errors.NotFoundError("Category not found")
	}

	// Authorization check.
	if user.Role == commontypes.RoleManagement {
		if appErr := guards.AuthorizeRestaurantOwner(ctx, s.restaurantRepo, category.RestaurantID.String(), user.UserID); appErr != nil {
			return appErr
		}
	}

	err = s.repo.Delete(ctx, id)
	if err != nil {
		return errors.InternalError(err)
	}

	// Invalidate Menu Cache.
	utils.InvalidateCacheByPrefix(ctx, "menus:list:")

	// Publish Event.
	s.publishCategoryEvent(ctx, id, "category.deleted")

	return nil
}

func (s *CategoryService) publishCategoryEvent(ctx context.Context, id string, topic string) {
	producer := events.GetGlobalProducer()
	if producer == nil {
		return
	}
	event := NewCategoryEvent(id, topic)
	if err := producer.Publish(ctx, event); err != nil {
		logger.Log.Error("Failed to publish category event", zap.String("topic", topic), zap.Error(err))
	}
}

