package services

import (
	"context"
	"io"
	"path/filepath"
	"strings"
	"time"

	"github.com/alibaba0010/postgres-api/internal/common/errors"
	"github.com/alibaba0010/postgres-api/internal/common/s3"
	"github.com/alibaba0010/postgres-api/internal/restaurants/dto"
	"github.com/alibaba0010/postgres-api/internal/restaurants/models"
	"github.com/alibaba0010/postgres-api/internal/restaurants/repositories"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/google/uuid"
)

// generateKey determines duplicate S3 key logic
func generateKey(s3Service *s3.S3Service, userID, filename, contentType string) string {
	if strings.HasPrefix(contentType, "video/") {
		ext := filepath.Ext(filename)
		return s3Service.GetVideoUploadKey(ext)
	}
	return s3Service.GetMenuImageKey(userID, filename)
}

// GetMenuUploadURL generates a presigned URL for menu media uploads
func GetMenuUploadURL(ctx context.Context, userID string, filename string, contentType string) (string, string, error) {
	s3Service, err := s3.NewS3Service()
	if err != nil {
		return "", "", err
	}

	key := generateKey(s3Service, userID, filename, contentType)

	url, err := s3Service.GenerateUploadURL(ctx, key, contentType)
	if err != nil {
		return "", "", err
	}

	// Also return the final public URL (CloudFront)
	publicURL := s3Service.GetCloudFrontURL(key)

	return url, publicURL, nil
}

// UploadMenuMedia handles direct media upload to S3
func UploadMenuMedia(ctx context.Context, userID string, filename string, contentType string, body io.Reader) (string, error) {
	s3Service, err := s3.NewS3Service()
	if err != nil {
		return "", err
	}

	key := generateKey(s3Service, userID, filename, contentType)

	err = s3Service.DirectUpload(ctx, key, body, contentType)
	if err != nil {
		return "", err
	}

	return s3Service.GetCloudFrontURL(key), nil
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

// InitiateMultipartUpload starts a multipart upload and returns details
func InitiateMultipartUpload(ctx context.Context, userID, filename, contentType string) (*dto.InitiateMultipartUploadResponse, error) {
	s3Service, err := s3.NewS3Service()
	if err != nil {
		return nil, err
	}

	key := generateKey(s3Service, userID, filename, contentType)

	uploadID, err := s3Service.InitiateMultipartUpload(ctx, key, contentType)
	if err != nil {
		return nil, err
	}

	return &dto.InitiateMultipartUploadResponse{
		UploadID: uploadID,
		Key:      key,
	}, nil
}

// GeneratePartPresignedURL generates a presigned URL for a specific part
func GeneratePartPresignedURL(ctx context.Context, key, uploadID string, partNumber int32) (string, error) {
	s3Service, err := s3.NewS3Service()
	if err != nil {
		return "", err
	}

	return s3Service.GeneratePresignPartURL(ctx, key, uploadID, partNumber)
}

// CompleteMultipartUpload completes the multipart upload
func CompleteMultipartUpload(ctx context.Context, key, uploadID string, parts []dto.CompletedPart) (string, error) {
	s3Service, err := s3.NewS3Service()
	if err != nil {
		return "", err
	}

	// Convert DTO parts to S3 types
	s3Parts := make([]types.CompletedPart, len(parts))
	for i, p := range parts {
		s3Parts[i] = types.CompletedPart{
			PartNumber: aws.Int32(p.PartNumber),
			ETag:       aws.String(p.ETag),
		}
	}

	return s3Service.CompleteMultipartUpload(ctx, key, uploadID, s3Parts)
}
