package services

import (
	"context"
	"mime/multipart"
	"time"

	"github.com/alibaba0010/postgres-api/internal/common/errors"
	"github.com/alibaba0010/postgres-api/internal/common/s3"
	"github.com/alibaba0010/postgres-api/internal/database"
	"github.com/alibaba0010/postgres-api/internal/restaurants/dto"
	"github.com/alibaba0010/postgres-api/internal/restaurants/models"
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

	_, err = database.DB.NewInsert().Model(menu).Exec(ctx)
	if err != nil {
		return nil, errors.InternalError(err)
	}

	return &dto.MenuResponse{
		ID:              menu.ID.String(),
		Name:            menu.Name,
		Description:     menu.Description,
		Price:           menu.Price,
		ImageURLs:       menu.ImageURLs,
		VideoURL:        menu.VideoURL,
		RestaurantID:    menu.RestaurantID.String(),
		IsAvailable:     menu.IsAvailable,
		PrepTimeMinutes: menu.PrepTimeMinutes,
		Calories:        menu.Calories,
		CreatedAt:       menu.CreatedAt.Format(time.RFC3339),
		UpdatedAt:       menu.UpdatedAt.Format(time.RFC3339),
	}, nil
}
