package services

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/alibaba0010/postgres-api/internal/common/errors"
	"github.com/alibaba0010/postgres-api/internal/common/events"
	"github.com/alibaba0010/postgres-api/internal/common/logger"
	"github.com/alibaba0010/postgres-api/internal/common/s3"
	commontypes "github.com/alibaba0010/postgres-api/internal/common/types"
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

// MenuService provides business logic for menu operations
type MenuService struct {
	repo           *repositories.MenuRepository
	restaurantRepo *repositories.RestaurantRepository
	s3Service      *s3.S3Service
}

type MenuServiceInterface interface{
	
}

// NewMenuService creates and returns a new menu service instance
func NewMenuService(menuRepo *repositories.MenuRepository, restaurantRepo *repositories.RestaurantRepository, s3Service *s3.S3Service) *MenuService {
	return &MenuService{
		repo:           menuRepo,
		restaurantRepo: restaurantRepo,
		s3Service:      s3Service,
	}
}

// isValidMediaType checks if the file extension is allowed
func isValidMediaType(filename string) bool {
	ext := strings.ToLower(filepath.Ext(filename))
	validExts := map[string]bool{
		".jpg": true, ".jpeg": true, ".png": true, ".webp": true, ".gif": true, ".avif": true, ".heic": true,
		".mp4": true, ".mov": true, ".avi": true, ".webm": true, ".m4v": true, ".mkv": true,
	}
	return validExts[ext]
}

// isAllowedContentType checks if the MIME type is in the allowed list
func isAllowedContentType(contentType string) bool {
	allowed := map[string]bool{
		"image/jpeg": true, "image/png": true, "image/webp": true, "image/gif": true, "image/avif": true,
		"video/mp4": true, "video/quicktime": true, "video/x-msvideo": true, "video/webm": true, "video/x-matroska": true,
	}
	return allowed[contentType]
}

// generateKey determines duplicate S3 key logic
func generateKey(s3Service *s3.S3Service, userID, filename, contentType string) string {
	ext := filepath.Ext(filename)
	if strings.HasPrefix(contentType, "video/") {
		return s3Service.GetVideoUploadKey(ext)
	}
	return s3Service.GetMenuImageKey(userID, filename)
}

// MapToResponse maps a Menu model to MenuResponse DTO
func (ms *MenuService) MapToResponse(m *models.Menu) *dto.MenuResponse {
	resp := &dto.MenuResponse{
		ID:              m.ID.String(),
		Name:            m.Name,
		Description:     m.Description,
		Price:           m.Price,
		ImageURLs:       m.ImageURLs,
		VideoURL:        m.VideoURL,
		RestaurantID:    m.RestaurantID.String(),
		Tags:            m.Tags,
		IsAvailable:     m.IsAvailable,
		PrepTimeMinutes: m.PrepTimeMinutes,
		Calories:        m.Calories,
		CreatedAt:       m.CreatedAt.Format(time.RFC3339),
		UpdatedAt:       m.UpdatedAt.Format(time.RFC3339),
	}
	if m.CategoryID != nil {
		resp.CategoryID = m.CategoryID.String()
	}
	return resp
}

// InitiateMultipartUpload starts a multipart upload and returns uploadid and unique key 
func (ms *MenuService) InitiateMultipartUpload(ctx context.Context, userID, filename, contentType string) (*dto.InitiateMultipartUploadResponse, error) {
	// Recommendation: Restrict file types explicitly
	if !isAllowedContentType(contentType) {
		return nil, errors.ValidationError("Invalid content type")
	}

	key := generateKey(ms.s3Service, userID, filename, contentType)

	uploadID, err := ms.s3Service.InitiateMultipartUpload(ctx, key, contentType)
	if err != nil {
		return nil, err
	}

	return &dto.InitiateMultipartUploadResponse{
		UploadID: uploadID,
		Key:      key,
	}, nil
}

// GetPartPresignedURL generates a presigned URL for a specific part
func (ms *MenuService) GetPartPresignedURL(ctx context.Context, key, uploadID string, partNumber int32) (string, error) {
	return ms.s3Service.GetPresignedPartURL(ctx, key, uploadID, partNumber)
}

// CompleteMultipartUpload completes the multipart upload
func (ms *MenuService) CompleteMultipartUpload(ctx context.Context, key, uploadID string, parts []dto.CompletedPart) (string, error) {
	// Convert DTO parts to S3 types
	s3Parts := make([]types.CompletedPart, len(parts))
	for i, p := range parts {
		s3Parts[i] = types.CompletedPart{
			PartNumber: aws.Int32(p.PartNumber),
			ETag:       aws.String(p.ETag),
		}
	}

	return ms.s3Service.CompleteMultipartUpload(ctx, key, uploadID, s3Parts)
}

// GetUploadURL generates a presigned URL for menu media uploads
func (ms *MenuService) GetUploadURL(ctx context.Context, userID string, filename string, contentType string) (string, string, *errors.AppError) {
	// Recommendation: Never trust the file extension
	if !isValidMediaType(filename) {
		return "", "", errors.ValidationError("Invalid file extension")
	}

	// Recommendation: Restrict file types explicitly
	if !isAllowedContentType(contentType) {
		return "", "", errors.ValidationError("Invalid content type")
	}

	key := generateKey(ms.s3Service, userID, filename, contentType)

	url, err := ms.s3Service.GenerateUploadURL(ctx, key, contentType)
	if err != nil {
		return "", "", errors.InternalError(err)
	}

	// Also return the final public URL (CloudFront)
	publicURL := ms.s3Service.GetCloudFrontURL(key)

	return url, publicURL, nil
}

// UploadMedia handles direct media upload to S3
func (ms *MenuService) UploadMedia(ctx context.Context, userID string, filename string, contentType string, body io.Reader) (string, *errors.AppError) {
	// Recommendation: Never trust the file extension
	if !isValidMediaType(filename) {
		return "", errors.ValidationError("Invalid file extension")
	}

	// Recommendation: Validate MIME types from content (Sniffing)
	// Read first 512 bytes to detect content type
	buffer := make([]byte, 512)
	n, err := body.Read(buffer)
	if err != nil && err != io.EOF {
		return "", errors.InternalError(err)
	}
	detectedType := http.DetectContentType(buffer[:n])

	// Recommendation: Restrict file types explicitly
	if !isAllowedContentType(detectedType) {
		logger.Log.Warn("Restricted file type attempt", zap.String("detected", detectedType), zap.String("user_id", userID))
		return "", errors.ValidationError("Invalid file content type")
	}

	// Create a new reader combining the buffer and the rest of the body
	fullBody := io.MultiReader(strings.NewReader(string(buffer[:n])), body)

	key := generateKey(ms.s3Service, userID, filename, detectedType)

	err = ms.s3Service.DirectUpload(ctx, key, fullBody, detectedType)
	if err != nil {
		return "", errors.InternalError(err)
	}

	return ms.s3Service.GetCloudFrontURL(key), nil
}

// AuthorizeRestaurantOwner checks if the user has permission to manage the restaurant
func (ms *MenuService) AuthorizeRestaurantOwner(ctx context.Context, restaurantID string, userID string) *errors.AppError {
	restaurant, err := ms.restaurantRepo.FindByID(ctx, restaurantID)
	if err != nil {
		return errors.NotFoundError("Restaurant not found")
	}
	if restaurant.UserID == nil || restaurant.UserID.String() != userID {
		return errors.ForbiddenError("You do not have permission to manage this restaurant's resources")
	}
	return nil
}

// Create creates a new menu item
func (ms *MenuService) Create(ctx context.Context, input dto.CreateMenuInput, userID string, userRole commontypes.UserRole) (*dto.MenuResponse, *errors.AppError) {
	// Permission logic: Managers can only add to their own restaurants
	if userRole == commontypes.RoleManagement {
		if err := ms.AuthorizeRestaurantOwner(ctx, input.RestaurantID, userID); err != nil {
			return nil, err
		}
	}

	// Validate input
	if err := utils.ValidateInput(input); err != nil {
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

	menu := &models.Menu{
		Name:            input.Name,
		Description:     input.Description,
		Price:           input.Price,
		ImageURLs:       input.ImageURLs,
		VideoURL:        input.VideoURL,
		RestaurantID:    restaurantID,
		Tags:            input.Tags,
		IsAvailable:     input.IsAvailable,
		PrepTimeMinutes: input.PrepTimeMinutes,
		Calories:        input.Calories,
	}

	if input.CategoryID != "" {
		cID, err := uuid.Parse(input.CategoryID)
		if err == nil {
			menu.CategoryID = &cID
		}
	}

	err = ms.repo.Create(ctx, menu)
	if err != nil {
		return nil, errors.InternalError(err)
	}

	return ms.MapToResponse(menu), nil
}

// GetByID retrieves a menu item by ID
func (ms *MenuService) GetByID(ctx context.Context, id string) (*models.Menu, *errors.AppError) {
	menu, err := ms.repo.FindByID(ctx, id)
	if err != nil {
		return nil, errors.NotFoundError("Menu item not found")
	}
	return menu, nil
}

// ListMenus retrieves a paginated list of menu items with filters/cache
func (ms *MenuService) ListMenus(ctx context.Context, limit int, cursor string, queryStr string, restaurantID string, categoryID string, tags []string, minPrice, maxPrice *float64, isAvailable *bool, sortBy, order string) ([]dto.MenuResponse, string, bool, int64, *errors.AppError) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}

	// Generate Cache Key
	cacheKeyPayload := fmt.Sprintf("%d:%s:%s:%s:%s:%v:%v:%v:%v:%s:%s", limit, cursor, queryStr, restaurantID, categoryID, tags, minPrice, maxPrice, isAvailable, sortBy, order)
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
	menus, nextCursor, hasMore, total, err := ms.repo.FindAll(ctx, limit, cursor, queryStr, restaurantID, categoryID, tags, minPrice, maxPrice, isAvailable, sortBy, order)
	if err != nil {
		return nil, "", false, 0, errors.InternalError(err)
	}

	responses := make([]dto.MenuResponse, len(menus))
	for i, m := range menus {
		responses[i] = *ms.MapToResponse(&m)
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

// Update updates an existing menu item
func (ms *MenuService) Update(ctx context.Context, id string, input dto.UpdateMenuInput, userID string, userRole commontypes.UserRole) (*dto.MenuResponse, *errors.AppError) {
	menu, appErr := ms.GetByID(ctx, id)
	if appErr != nil {
		return nil, appErr
	}

	// Permission logic: Managers can only update their own items
	if userRole == commontypes.RoleManagement {
		if err := ms.AuthorizeRestaurantOwner(ctx, menu.RestaurantID.String(), userID); err != nil {
			return nil, err
		}
	}

	// Validate input
	if err := utils.ValidateInput(input); err != nil {
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
	if input.Tags != nil {
		menu.Tags = input.Tags
	}
	if input.CategoryID != nil {
		cID, err := uuid.Parse(*input.CategoryID)
		if err == nil {
			menu.CategoryID = &cID
		}
	}
	if input.PrepTimeMinutes != nil {
		menu.PrepTimeMinutes = *input.PrepTimeMinutes
	}
	if input.Calories != nil {
		menu.Calories = *input.Calories
	}

	err := ms.repo.Update(ctx, nil, menu)
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

	return ms.MapToResponse(menu), nil
}



