package services

import (
	"context"
	"time"

	commondto "github.com/alibaba0010/postgres-api/internal/common/dto"
	"github.com/alibaba0010/postgres-api/internal/common/errors"
	"github.com/alibaba0010/postgres-api/internal/common/guards"
	commontypes "github.com/alibaba0010/postgres-api/internal/common/types"
	"github.com/alibaba0010/postgres-api/internal/restaurants/dto"
	"github.com/alibaba0010/postgres-api/internal/restaurants/models"
	"github.com/alibaba0010/postgres-api/internal/restaurants/repositories"
	"github.com/alibaba0010/postgres-api/internal/utils"
	"github.com/google/uuid"
)

// CategoryService provides business logic for menu category operations.
type CategoryService struct {
	repo           *repositories.CategoryRepository
	restaurantRepo *repositories.RestaurantRepository
}

// CategoryServiceInterface defines the interface for category operations.
type CategoryServiceInterface interface {
	CreateCategory(ctx context.Context, user *commondto.AuthenticatedUser, input dto.CreateCategoryInput) (*dto.CategoryResponse, *errors.AppError)
	ListCategoriesByRestaurant(ctx context.Context, restaurantID string) ([]dto.CategoryResponse, *errors.AppError)
}

// NewCategoryService creates a new instance of CategoryService.
func NewCategoryService(repo *repositories.CategoryRepository, restaurantRepo *repositories.RestaurantRepository) *CategoryService {
	return &CategoryService{
		repo:           repo,
		restaurantRepo: restaurantRepo,
	}
}

// MapCategoryToResponse transforms a MenuCategory model into a CategoryResponse DTO.
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

// CreateCategory handles the logic for creating a new menu category.
func (s *CategoryService) CreateCategory(ctx context.Context, user *commondto.AuthenticatedUser, input dto.CreateCategoryInput) (*dto.CategoryResponse, *errors.AppError) {
	// Authorization check: User must own the restaurant
	if user.Role == commontypes.RoleManagement {
		if appErr := guards.AuthorizeRestaurantOwner(ctx, s.restaurantRepo, input.RestaurantID, user.UserID); appErr != nil {
			return nil, appErr
		}
	}

	if err := utils.ValidateInput(input); err != nil {
		return nil, err
	}

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

	err = s.repo.Create(ctx, category)
	if err != nil {
		return nil, errors.InternalError(err)
	}

	resp := s.MapCategoryToResponse(category)
	return &resp, nil
}

// ListCategoriesByRestaurant returns all categories associated with a given restaurant.
func (s *CategoryService) ListCategoriesByRestaurant(ctx context.Context, restaurantID string) ([]dto.CategoryResponse, *errors.AppError) {
	categories, err := s.repo.FindByRestaurantID(ctx, restaurantID)
	if err != nil {
		return nil, errors.InternalError(err)
	}

	responses := make([]dto.CategoryResponse, len(categories))
	for i, c := range categories {
		responses[i] = s.MapCategoryToResponse(&c)
	}

	return responses, nil
}

