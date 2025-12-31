package controllers

import (
	"context"
	"encoding/json"
	"net/http"
	"path/filepath"
	"strings"

	commondto "github.com/alibaba0010/postgres-api/internal/common/dto"
	"github.com/alibaba0010/postgres-api/internal/common/errors"
	"github.com/alibaba0010/postgres-api/internal/common/guards"
	"github.com/alibaba0010/postgres-api/internal/common/types"
	"github.com/alibaba0010/postgres-api/internal/restaurants/dto"
	"github.com/alibaba0010/postgres-api/internal/restaurants/services"
	"github.com/alibaba0010/postgres-api/internal/utils"
	"github.com/gorilla/mux"
)

// GetMenuUploadURLHandler handles the request for a presigned URL for menu media uploads
func GetMenuUploadURLHandler(writer http.ResponseWriter, request *http.Request) {
	filename := request.URL.Query().Get("filename")
	contentType := request.URL.Query().Get("content_type")

	if filename == "" || contentType == "" {
		errors.ErrorResponse(writer, request, errors.ValidationError("filename and content_type are required"))
		return
	}

	user := guards.ExtractAuthenticatedUser(request)
	if user == nil {
		errors.ErrorResponse(writer, request, errors.UnauthorizedError("Authentication required"))
		return
	}

	// Basic type validation
	if !isValidMediaType(filename) {
		errors.ErrorResponse(writer, request, errors.ValidationError("Invalid file type"))
		return
	}

	uploadURL, publicURL, err := services.GetMenuUploadURL(request.Context(), user.UserID, filename, contentType)
	if err != nil {
		errors.ErrorResponse(writer, request, errors.InternalError(err))
		return
	}

	utils.WriteJSON(writer, http.StatusOK, map[string]interface{}{
		"title": "Presigned URL generated successfully",
		"data": map[string]string{
			"upload_url": uploadURL,
			"public_url": publicURL,
		},
	})
}

// isValidMediaType checks if the file extension is allowed
func isValidMediaType(filename string) bool {
	ext := strings.ToLower(filepath.Ext(filename))
	validExts := map[string]bool{
		".jpg": true, ".jpeg": true, ".png": true, ".webp": true, ".gif": true, ".avif": true, ".heic": true,
		".mp4": true, ".mov": true, ".avi": true, ".webm": true, ".m4v": true, ".mkv": true,
		// ".jpg": true, ".jpeg": true, ".png": true, ".webp": true,
		// ".mp4": true, ".mov": true, ".avi": true,
	}
	return validExts[ext]
}

// UploadMenuMediaHandler handles direct media upload (Multipart Form)
func UploadMenuMediaHandler(writer http.ResponseWriter, request *http.Request) {
	// Parse the multipart form (max 50MB)
	err := request.ParseMultipartForm(50 << 20)
	if err != nil {
		errors.ErrorResponse(writer, request, errors.ValidationError("File too large or invalid form"))
		return
	}

	file, header, err := request.FormFile("file")
	if err != nil {
		errors.ErrorResponse(writer, request, errors.ValidationError("File is required"))
		return
	}
	defer file.Close()

	if !isValidMediaType(header.Filename) {
		errors.ErrorResponse(writer, request, errors.ValidationError("Invalid file type"))
		return
	}

	user := guards.ExtractAuthenticatedUser(request)
	if user == nil {
		errors.ErrorResponse(writer, request, errors.UnauthorizedError("Authentication required"))
		return
	}

	contentType := header.Header.Get("Content-Type")
	filename := header.Filename

	publicURL, appErr := services.UploadMenuMedia(request.Context(), user.UserID, filename, contentType, file)
	if appErr != nil {
		errors.ErrorResponse(writer, request, errors.InternalError(appErr))
		return
	}

	utils.WriteJSON(writer, http.StatusOK, map[string]string{
		"title": "Upload successful",
		"data":  publicURL,
	})
}

// InitiateMultipartUploadHandler handles the initiation of a multipart upload
func InitiateMultipartUploadHandler(writer http.ResponseWriter, request *http.Request) {
	var input dto.InitiateMultipartUploadInput
	if err := json.NewDecoder(request.Body).Decode(&input); err != nil {
		errors.ErrorResponse(writer, request, errors.ValidationError("Invalid request body"))
		return
	}

	user := guards.ExtractAuthenticatedUser(request)
	if user == nil {
		errors.ErrorResponse(writer, request, errors.UnauthorizedError("Authentication required"))
		return
	}

	resp, err := services.InitiateMultipartUpload(request.Context(), user.UserID, input.Filename, input.ContentType)
	if err != nil {
		errors.ErrorResponse(writer, request, errors.InternalError(err))
		return
	}

	utils.WriteJSON(writer, http.StatusCreated, map[string]interface{}{
		"title": "Multipart upload initiated",
		"data":  resp,
	})
}

// GenerateMultipartPartURLHandler handles generating a presigned URL for a part
func GenerateMultipartPartURLHandler(writer http.ResponseWriter, request *http.Request) {
	query := request.URL.Query()
	key := query.Get("key")
	uploadID := query.Get("upload_id")
	partNumberStr := query.Get("part_number")

	if key == "" || uploadID == "" || partNumberStr == "" {
		errors.ErrorResponse(writer, request, errors.ValidationError("key, upload_id, and part_number are required"))
		return
	}

	partNumber := utils.ParseInt(partNumberStr, 0)
	if partNumber <= 0 {
		errors.ErrorResponse(writer, request, errors.ValidationError("part_number must be a positive integer"))
		return
	}

	url, err := services.GeneratePartPresignedURL(request.Context(), key, uploadID, int32(partNumber))
	if err != nil {
		errors.ErrorResponse(writer, request, errors.InternalError(err))
		return
	}

	utils.WriteJSON(writer, http.StatusOK, map[string]interface{}{
		"title": "Presigned part URL generated",
		"data":  url,
	})
}

// CompleteMultipartUploadHandler handles the completion of a multipart upload
func CompleteMultipartUploadHandler(writer http.ResponseWriter, request *http.Request) {
	var input dto.CompleteMultipartUploadInput
	if err := json.NewDecoder(request.Body).Decode(&input); err != nil {
		errors.ErrorResponse(writer, request, errors.ValidationError("Invalid request body"))
		return
	}

	publicURL, err := services.CompleteMultipartUpload(request.Context(), input.Key, input.UploadID, input.Parts)
	if err != nil {
		errors.ErrorResponse(writer, request, errors.InternalError(err))
		return
	}

	utils.WriteJSON(writer, http.StatusOK, map[string]interface{}{
		"title": "Multipart upload completed",
		"data":  map[string]string{"url": publicURL},
	})
}



// verifyRestaurantOwnership checks if the user owns the restaurant
func verifyRestaurantOwnership(ctx context.Context, restaurantID string, userID string) *errors.AppError {
	restaurant, err := services.GetRestaurantByID(ctx, restaurantID)
	if err != nil {
		return err
	}
	if restaurant.UserID == nil || *restaurant.UserID != userID {
		return errors.ForbiddenError("You do not have permission to manage this restaurant's resources")
	}
	return nil
}

// CreateMenuHandler handles the creation of a menu item
func CreateMenuHandler(writer http.ResponseWriter, request *http.Request) {
	var input dto.CreateMenuInput
	if err := json.NewDecoder(request.Body).Decode(&input); err != nil {
		errors.ErrorResponse(writer, request, errors.ValidationError("Invalid request body"))
		return
	}

	user := guards.ExtractAuthenticatedUser(request)
	if user == nil {
		errors.ErrorResponse(writer, request, errors.UnauthorizedError("Authentication required"))
		return
	}

	// Permission logic: Managers can only add to their own restaurants
	if user.Role == types.RoleManagement {
		if err := verifyRestaurantOwnership(request.Context(), input.RestaurantID, user.UserID); err != nil {
			errors.ErrorResponse(writer, request, err)
			return
		}
	}

	menu, appErr := services.CreateMenu(request.Context(), input)
	if appErr != nil {
		errors.ErrorResponse(writer, request, appErr)
		return
	}

	utils.WriteJSON(writer, http.StatusCreated, map[string]any{
		"title": "Menu item created successfully",
		"data":  menu,
	})
}

// GetMenuHandler handles retrieving a single menu item
func GetMenuHandler(writer http.ResponseWriter, request *http.Request) {
	vars := mux.Vars(request)
	id := vars["id"]

	menu, appErr := services.GetMenuByID(request.Context(), id)
	if appErr != nil {
		errors.ErrorResponse(writer, request, appErr)
		return
	}

	utils.WriteJSON(writer, http.StatusOK, map[string]any{
		"data": services.MapMenuToResponse(menu),
	})
}

// ListMenusHandler handles listing and filtering menu items
func ListMenusHandler(writer http.ResponseWriter, request *http.Request) {
	params := utils.ParseListParams(request)
	query := request.URL.Query()
	restaurantID := query.Get("restaurant_id")

	var minPrice, maxPrice *float64
	if mp := query.Get("min_price"); mp != "" {
		p := utils.ParseFloat(mp, 0)
		minPrice = &p
	}
	if mp := query.Get("max_price"); mp != "" {
		p := utils.ParseFloat(mp, 0)
		maxPrice = &p
	}

	var isAvailable *bool
	if ia := query.Get("is_available"); ia != "" {
		b := ia == "true"
		isAvailable = &b
	}

	menus, total, appErr := services.ListMenus(request.Context(), params.Page, params.PageSize, params.Query, restaurantID, minPrice, maxPrice, isAvailable, params.SortBy, params.Order)
	if appErr != nil {
		errors.ErrorResponse(writer, request, appErr)
		return
	}

	utils.WriteJSON(writer, http.StatusOK, dto.MenusListResponse{
		Data: menus,
		Meta: commondto.PaginationMeta{
			Page:       params.Page,
			PageSize:   params.PageSize,
			Total:      total,
			TotalPages: utils.CalculateTotalPages(total, params.PageSize),
		},
	})
}

// UpdateMenuHandler handles updating an existing menu item
func UpdateMenuHandler(writer http.ResponseWriter, request *http.Request) {
	vars := mux.Vars(request)
	id := vars["id"]

	var input dto.UpdateMenuInput
	if err := json.NewDecoder(request.Body).Decode(&input); err != nil {
		errors.ErrorResponse(writer, request, errors.ValidationError("Invalid request body"))
		return
	}

	user := guards.ExtractAuthenticatedUser(request)
	if user == nil {
		errors.ErrorResponse(writer, request, errors.UnauthorizedError("Authentication required"))
		return
	}

	// Fetch existing menu to check ownership
	menuObj, appErr := services.GetMenuByID(request.Context(), id)
	if appErr != nil {
		errors.ErrorResponse(writer, request, appErr)
		return
	}

	// Permission logic: Managers can only update their own items
	if user.Role == types.RoleManagement {
		if err := verifyRestaurantOwnership(request.Context(), menuObj.RestaurantID.String(), user.UserID); err != nil {
			errors.ErrorResponse(writer, request, err)
			return
		}
	}

	updatedMenu, appErr := services.UpdateMenu(request.Context(), id, input)
	if appErr != nil {
		errors.ErrorResponse(writer, request, appErr)
		return
	}

	utils.WriteJSON(writer, http.StatusOK, map[string]any{
		"title": "Menu item updated successfully",
		"data":  updatedMenu,
	})
}
