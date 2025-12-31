package services

import (
	"context"
	"mime/multipart"
	"time"

	"github.com/alibaba0010/postgres-api/internal/common/errors"
	"github.com/alibaba0010/postgres-api/internal/common/s3"
	"github.com/alibaba0010/postgres-api/internal/restaurants/dto"
	"github.com/alibaba0010/postgres-api/internal/restaurants/models"
	"github.com/alibaba0010/postgres-api/internal/restaurants/repositories"
	"github.com/google/uuid"
)

// UploadMenuMedia uploads a file to S3
func UploadMenuMedia(file multipart.File, header *multipart.FileHeader) (string, error) {
	s3Service, err := s3.NewS3Service()
	if err != nil {
		return "", err
	}
	return s3Service.UploadFile(file, header, "menus")
}

// CreateMenu creates a new menu item
func CreateMenu(ctx context.Context, input dto.CreateMenuInput) (*dto.MenuResponse, *errors.AppError) {
	restaurantID, err := uuid.Parse(input.RestaurantID)
	if err != nil {
		return nil, errors.ValidationError("Invalid restaurant ID")
	}

	menu := &models.Menu{
		Name:            input.Name,
		Description:     input.Description,
		Price:           input.Price,
		ImageURLs:       input.ImageURLs,
		VideoURL:        input.VideoURL,
		RestaurantID:    restaurantID,
		IsAvailable:     input.IsAvailable,
		PrepTimeMinutes: input.PrepTimeMinutes,
		Calories:        input.Calories,
	}

	err = repositories.MenuRepo.Create(ctx, menu)
	if err != nil {
		return nil, errors.InternalError(err)
	}

	return MapMenuToResponse(menu), nil
}

// GetMenuByID retrieves a menu item by ID
func GetMenuByID(ctx context.Context, id string) (*models.Menu, *errors.AppError) {
	menu, err := repositories.MenuRepo.FindByID(ctx, id)
	if err != nil {
		return nil, errors.NotFoundError("Menu item not found")
	}
	return menu, nil
}

// ListMenus retrieves a paginated list of menu items with filters
func ListMenus(ctx context.Context, page, pageSize int, queryStr string, restaurantID string, minPrice, maxPrice *float64, isAvailable *bool, sortBy, order string) ([]dto.MenuResponse, int64, *errors.AppError) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 20
	}

	menus, total, err := repositories.MenuRepo.FindAll(ctx, page, pageSize, queryStr, restaurantID, minPrice, maxPrice, isAvailable, sortBy, order)
	if err != nil {
		return nil, 0, errors.InternalError(err)
	}

	responses := make([]dto.MenuResponse, len(menus))
	for i, m := range menus {
		responses[i] = *MapMenuToResponse(&m)
	}

	return responses, total, nil
}

// UpdateMenu updates an existing menu item
func UpdateMenu(ctx context.Context, id string, input dto.UpdateMenuInput) (*dto.MenuResponse, *errors.AppError) {
	menu, appErr := GetMenuByID(ctx, id)
	if appErr != nil {
		return nil, appErr
	}

	if input.Name != nil {
		menu.Name = *input.Name
	}
	if input.Description != nil {
		menu.Description = *input.Description
	}
	if input.Price != nil {
		menu.Price = *input.Price
	}
	if input.ImageURLs != nil {
		menu.ImageURLs = input.ImageURLs
	}
	if input.VideoURL != nil {
		menu.VideoURL = *input.VideoURL
	}
	if input.IsAvailable != nil {
		menu.IsAvailable = *input.IsAvailable
	}
	if input.PrepTimeMinutes != nil {
		menu.PrepTimeMinutes = *input.PrepTimeMinutes
	}
	if input.Calories != nil {
		menu.Calories = *input.Calories
	}

	err := repositories.MenuRepo.Update(ctx, menu)
	if err != nil {
		return nil, errors.InternalError(err)
	}

	return MapMenuToResponse(menu), nil
}

// MapMenuToResponse maps a Menu model to MenuResponse DTO
func MapMenuToResponse(m *models.Menu) *dto.MenuResponse {
	return &dto.MenuResponse{
		ID:              m.ID.String(),
		Name:            m.Name,
		Description:     m.Description,
		Price:           m.Price,
		ImageURLs:       m.ImageURLs,
		VideoURL:        m.VideoURL,
		RestaurantID:    m.RestaurantID.String(),
		IsAvailable:     m.IsAvailable,
		PrepTimeMinutes: m.PrepTimeMinutes,
		Calories:        m.Calories,
		CreatedAt:       m.CreatedAt.Format(time.RFC3339),
		UpdatedAt:       m.UpdatedAt.Format(time.RFC3339),
	}
}
