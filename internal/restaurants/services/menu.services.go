package services

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"time"

	"github.com/alibaba0010/postgres-api/internal/common/errors"
	"github.com/alibaba0010/postgres-api/internal/common/events"
	"github.com/alibaba0010/postgres-api/internal/common/logger"
	"github.com/alibaba0010/postgres-api/internal/common/s3"
	"github.com/alibaba0010/postgres-api/internal/database"
	"github.com/alibaba0010/postgres-api/internal/restaurants/dto"
	"github.com/alibaba0010/postgres-api/internal/restaurants/models"
	"github.com/alibaba0010/postgres-api/internal/restaurants/repositories"
	"github.com/alibaba0010/postgres-api/internal/utils"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// generateKey determines duplicate S3 key logic
func generateKey(s3Service *s3.S3Service, userID, filename, contentType string) string {
	if strings.HasPrefix(contentType, "video/") {
		ext := filepath.Ext(filename)
		return s3Service.GetVideoUploadKey(ext)
	}
	return s3Service.GetMenuImageKey(userID, filename)
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

// InitiateMultipartUpload starts a multipart upload and returns uploadid and unique key 
func InitiateMultipartUpload(ctx context.Context, userID, filename, contentType string) (*dto.InitiateMultipartUploadResponse, error) {
	s3Service, err := s3.NewS3Service(ctx)
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

// GetPartPresignedURL generates a presigned URL for a specific part
func GetPartPresignedURL(ctx context.Context, key, uploadID string, partNumber int32) (string, error) {
	s3Service, err := s3.NewS3Service(ctx)
	if err != nil {
		return "", err
	}

	return s3Service.GetPresignedPartURL(ctx, key, uploadID, partNumber)
}

// CompleteMultipartUpload completes the multipart upload
func CompleteMultipartUpload(ctx context.Context, key, uploadID string, parts []dto.CompletedPart) (string, error) {
	s3Service, err := s3.NewS3Service(ctx)
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
// GetMenuUploadURL generates a presigned URL for menu media uploads
func GetMenuUploadURL(ctx context.Context, userID string, filename string, contentType string) (string, string, error) {
	s3Service, err := s3.NewS3Service(ctx)
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
	s3Service, err := s3.NewS3Service(ctx)
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
	// Validate input
	if err := utils.ValidateAndError(input); err != nil {
		return nil, err
	}

	restaurantID, err := uuid.Parse(input.RestaurantID)
	if err != nil {
		return nil, errors.ValidationError("Invalid restaurant ID")
	}

	// Sanitize
	input.Name = strings.TrimSpace(input.Name)
	input.Description = strings.TrimSpace(input.Description)
	input.VideoURL = strings.TrimSpace(input.VideoURL)
	// ImageURLs are usually signed URLs or specific paths, but we can trim them too if needed.

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

// ListMenus retrieves a paginated list of menu items with filters/cache
func ListMenus(ctx context.Context, limit int, cursor string, queryStr string, restaurantID string, minPrice, maxPrice *float64, isAvailable *bool, sortBy, order string) ([]dto.MenuResponse, string, bool, int64, *errors.AppError) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}

	// Generate Cache Key
	cacheKeyPayload := fmt.Sprintf("%d:%s:%s:%s:%v:%v:%v:%s:%s", limit, cursor, queryStr, restaurantID, minPrice, maxPrice, isAvailable, sortBy, order)
	hash := sha256.Sum256([]byte(cacheKeyPayload))
	cacheKey := "menus:list:" + hex.EncodeToString(hash[:])

	// 1. Try Cache
	if database.RedisClient != nil {
		val, err := database.RedisClient.Get(ctx, cacheKey).Result()
		if err == nil {
			var cachedResult struct {
				Menus      []dto.MenuResponse `json:"menus"`
				NextCursor string             `json:"next_cursor"`
				HasMore    bool               `json:"has_more"`
				Total      int64              `json:"total"`
			}
			if err := json.Unmarshal([]byte(val), &cachedResult); err == nil {
				// Cache Hit
				return cachedResult.Menus, cachedResult.NextCursor, cachedResult.HasMore, cachedResult.Total, nil
			}
		}
	}

	// 2. Fetch from Repo
	menus, nextCursor, hasMore, total, err := repositories.MenuRepo.FindAll(ctx, limit, cursor, queryStr, restaurantID, minPrice, maxPrice, isAvailable, sortBy, order)
	if err != nil {
		return nil, "", false, 0, errors.InternalError(err)
	}

	responses := make([]dto.MenuResponse, len(menus))
	for i, m := range menus {
		responses[i] = *MapMenuToResponse(&m)
	}

	// 3. Set Cache
	if database.RedisClient != nil {
		cachedResult := struct {
			Menus      []dto.MenuResponse `json:"menus"`
			NextCursor string             `json:"next_cursor"`
			HasMore    bool               `json:"has_more"`
			Total      int64              `json:"total"`
		}{
			Menus:      responses,
			NextCursor: nextCursor,
			HasMore:    hasMore,
			Total:      total,
		}
		if data, err := json.Marshal(cachedResult); err == nil {
			database.RedisClient.Set(ctx, cacheKey, data, 5*time.Minute)
		}
	}

	return responses, nextCursor, hasMore, total, nil
}

// UpdateMenu updates an existing menu item
func UpdateMenu(ctx context.Context, id string, input dto.UpdateMenuInput) (*dto.MenuResponse, *errors.AppError) {
	menu, appErr := GetMenuByID(ctx, id)
	if appErr != nil {
		return nil, appErr
	}

	// Validate input
	if err := utils.ValidateAndError(input); err != nil {
		return nil, err
	}

	if input.Name != nil {
		menu.Name = strings.TrimSpace(*input.Name)
	}
	if input.Description != nil {
		menu.Description = strings.TrimSpace(*input.Description)
	}
	if input.Price != nil {
		menu.Price = *input.Price
	}
	if input.ImageURLs != nil {
		menu.ImageURLs = input.ImageURLs
	}
	if input.VideoURL != nil {
		menu.VideoURL = strings.TrimSpace(*input.VideoURL)
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

	// Publish Event
	producer := events.GetGlobalProducer()
	if producer != nil {
		event := NewMenuUpdatedEvent(menu.ID.String())
		if err := producer.Publish(ctx, event); err != nil {
			logger.Log.Error("Failed to publish menu updated event", zap.Error(err))
		}
	}

	return MapMenuToResponse(menu), nil
}



