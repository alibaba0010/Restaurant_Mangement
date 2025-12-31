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

// UploadMenuMediaHandler handles the upload of menu images/videos
func UploadMenuMediaHandler(writer http.ResponseWriter, request *http.Request) {
	// Limit 50MB
	request.ParseMultipartForm(50 << 20)

	file, header, err := request.FormFile("file")
	if err != nil {
		errors.ErrorResponse(writer, request, errors.ValidationError("File is required"))
		return
	}
	defer file.Close()

	// Basic type validation
	ext := strings.ToLower(filepath.Ext(header.Filename))
	validExts := map[string]bool{
		".jpg": true, ".jpeg": true, ".png": true, ".webp": true,
		".mp4": true, ".mov": true, ".avi": true,
	}
	if !validExts[ext] {
		errors.ErrorResponse(writer, request, errors.ValidationError("Invalid file type"))
		return
	}

	url, err := services.UploadMenuMedia(file, header)
	if err != nil {
		// Log error if needed, but return generic internal error
		errors.ErrorResponse(writer, request, errors.InternalError(err))
		return
	}

	utils.WriteJSON(writer, http.StatusOK, map[string]interface{}{
		"title": "File uploaded successfully",
		"data": map[string]string{
			"url": url,
		},
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
