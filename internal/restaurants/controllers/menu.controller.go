package controllers

import (
	"encoding/json"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/alibaba0010/postgres-api/internal/common/errors"
	"github.com/alibaba0010/postgres-api/internal/common/guards"
	"github.com/alibaba0010/postgres-api/internal/common/types"
	"github.com/alibaba0010/postgres-api/internal/restaurants/dto"
	"github.com/alibaba0010/postgres-api/internal/restaurants/services"
	"github.com/alibaba0010/postgres-api/internal/utils"
)

// UploadMenuMediaHandler handles the upload of menu images/videos
func UploadMenuMediaHandler(w http.ResponseWriter, r *http.Request) {
	// Limit 50MB
	r.ParseMultipartForm(50 << 20)

	file, header, err := r.FormFile("file")
	if err != nil {
		errors.ErrorResponse(w, r, errors.ValidationError("File is required"))
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
		errors.ErrorResponse(w, r, errors.ValidationError("Invalid file type"))
		return
	}

	url, err := services.UploadMenuMedia(file, header)
	if err != nil {
		// Log error if needed, but return generic internal error
		errors.ErrorResponse(w, r, errors.InternalError(err))
		return
	}

	utils.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"title": "File uploaded successfully",
		"data": map[string]string{
			"url": url,
		},
	})
}

// CreateMenuHandler handles the creation of a menu item
func CreateMenuHandler(w http.ResponseWriter, r *http.Request) {
	var input dto.CreateMenuInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		errors.ErrorResponse(w, r, errors.ValidationError("Invalid request body"))
		return
	}

	// Permission logic regarding restaurant ownership
	user := guards.ExtractAuthenticatedUser(r)
	if user == nil {
		errors.ErrorResponse(w, r, errors.UnauthorizedError("User not authenticated"))
		return
	}

	if user.Role == types.RoleManagement {
		// Verify ownership
		restaurant, err := services.GetRestaurantByID(r.Context(), input.RestaurantID)
		if err != nil {
			errors.ErrorResponse(w, r, err)
			return
		}
		
		if restaurant.UserID == nil || *restaurant.UserID != user.UserID {
			errors.ErrorResponse(w, r, errors.ForbiddenError("You do not have permission to add menu items to this restaurant"))
			return
		}
	}

	menu, appErr := services.CreateMenu(r.Context(), input)
	if appErr != nil {
		errors.ErrorResponse(w, r, appErr)
		return
	}

	utils.WriteJSON(w, http.StatusCreated, map[string]interface{}{
		"title": "Menu item created successfully",
		"data":  menu,
	})
}
