package services

import (
	"context"
	"time"

	"github.com/alibaba0010/postgres-api/internal/common/errors"
	"github.com/alibaba0010/postgres-api/internal/restaurants/dto"
	"github.com/alibaba0010/postgres-api/internal/restaurants/models"
	"github.com/alibaba0010/postgres-api/internal/restaurants/repositories"
	"github.com/alibaba0010/postgres-api/internal/utils"
	"github.com/google/uuid"
)

type CategoryService struct {
	repo *repositories.CategoryRepository
}
type CategoryServiceInterface interface{
	CreateCategory(ctx context.Context, input dto.CreateCategoryInput) (*dto.CategoryResponse, *errors.AppError)
	ListCategoriesByRestaurant(ctx context.Context, restaurantID string) ([]dto.CategoryResponse, *errors.AppError)
}
func NewCategoryService(repo *repositories.CategoryRepository) *CategoryService {
	return &CategoryService{repo: repo}
}

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

func (s *CategoryService) CreateCategory(ctx context.Context, input dto.CreateCategoryInput) (*dto.CategoryResponse, *errors.AppError) {
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
