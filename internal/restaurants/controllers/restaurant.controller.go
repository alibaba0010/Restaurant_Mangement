package controllers

import (
	"encoding/json"
	"net/http"

	commondto "github.com/alibaba0010/postgres-api/internal/common/dto"
	"github.com/alibaba0010/postgres-api/internal/common/errors"
	"github.com/alibaba0010/postgres-api/internal/common/guards"
	"github.com/alibaba0010/postgres-api/internal/common/types"
	"github.com/alibaba0010/postgres-api/internal/restaurants/dto"
	"github.com/alibaba0010/postgres-api/internal/restaurants/models"
	"github.com/alibaba0010/postgres-api/internal/restaurants/services"
	"github.com/alibaba0010/postgres-api/internal/utils"
	"github.com/gorilla/mux"
)

// CreateRestaurantHandler handles the creation of a new restaurant
func CreateRestaurantHandler(writer http.ResponseWriter, request *http.Request) {
	var input dto.CreateRestaurantInput
	if err := json.NewDecoder(request.Body).Decode(&input); err != nil {
		errors.ErrorResponse(writer, request, errors.ValidationError("Invalid request body"))
		return
	}
	
	// Validate input
	if err := utils.ValidateAndError(input); err != nil {
		errors.ErrorResponse(writer, request, err)
		return
	}

	// Extract authenticated user from context (set by AuthMiddleware)
	user := guards.ExtractAuthenticatedUser(request)
	if user == nil {
		errors.ErrorResponse(writer, request, errors.UnauthorizedError("User not authenticated"))
		return
	}

	// Pass user info directly to service
	resp, err := services.CreateRestaurant(request.Context(), input, user)
	if err != nil {
		errors.ErrorResponse(writer, request, err)
		return
	}

	utils.WriteJSON(writer, http.StatusCreated, map[string]any{
		"title": "Created Restaurant Successfully",
		"data":  resp,
	})
}

// GetRestaurantHandler retrieves a restaurant by ID
func GetRestaurantHandler(writer http.ResponseWriter, request *http.Request) {
	vars := mux.Vars(request)
	id := vars["id"]

	resp, err := services.GetRestaurantByID(request.Context(), id)
	if err != nil {
		errors.ErrorResponse(writer, request, err)
		return
	}

	// Check ownership if management role
	if user := guards.ExtractAuthenticatedUser(request); user != nil {
		if user.Role == types.RoleManagement {
			if resp.UserID == nil {
				errors.ErrorResponse(writer, request, errors.ForbiddenError("You do not have permission to view this restaurant"))
				return
			}
			// Compare UUIDs
			if *resp.UserID != user.UserID {
				errors.ErrorResponse(writer, request, errors.ForbiddenError("You do not have permission to view this restaurant"))
				return
			}
		}
	}

	utils.WriteJSON(writer, http.StatusOK, map[string]interface{}{
		"title": "Retrieved Restaurant Successfully",
		"data":  resp,
	})
}

// ListRestaurantsHandler lists restaurants with pagination
func ListRestaurantsHandler(writer http.ResponseWriter, request *http.Request) {
	params := utils.ParseListParams(request)

	// Filter logic based on user role
	// Filter logic based on user role
	var filterUserID *string
	var filterStatus *string
	
	activeStatus := string(models.RestaurantStatusActive)

	user := guards.ExtractAuthenticatedUser(request)

	if user != nil {
		switch user.Role {
		case types.RoleManagement:
			// Management -> Only own restaurants, but can see all statuses (e.g. pending/blocked)
			filterUserID = &user.UserID
			filterStatus = nil 
		case types.RoleAdmin:
			// Admin -> All restaurants, all statuses
			filterUserID = nil
			filterStatus = nil
		case types.RoleUser:
			// User -> All restaurants, but ONLY Active
			filterUserID = nil
			filterStatus = &activeStatus
		default:
			// Fallback -> Active only
			filterUserID = nil
			filterStatus = &activeStatus
		}
	} else {
		// Unauthenticated -> All restaurants, Active only
		filterUserID = nil
		filterStatus = &activeStatus
	}

	data, nextCursor, hasMore, total, err := services.GetAllRestaurants(request.Context(), params.Limit, params.Cursor, params.Query, filterUserID, filterStatus, params.SortBy, params.Order)
	if err != nil {
		errors.ErrorResponse(writer, request, err)
		return
	}

	utils.WriteJSON(writer, http.StatusOK, dto.RestaurantsListResponse{
		Title: "Restaurants retrieved successfully",
		Data:  data,
		Meta: commondto.CursorMeta{
			NextCursor: nextCursor,
			HasMore:    hasMore,
			Total:      total,
		},
	})
}

// UpdateRestaurantHandler handles updating a restaurant
func UpdateRestaurantHandler(writer http.ResponseWriter, request *http.Request) {
	vars := mux.Vars(request)
	id := vars["id"]

	var input dto.UpdateRestaurantInput
	if err := json.NewDecoder(request.Body).Decode(&input); err != nil {
		errors.ErrorResponse(writer, request, errors.ValidationError("Invalid request body"))
		return
	}

	// Validate input
	if err := utils.ValidateAndError(input); err != nil {
		errors.ErrorResponse(writer, request, err)
		return
	}

	user := guards.ExtractAuthenticatedUser(request)
	if user == nil {
		errors.ErrorResponse(writer, request, errors.UnauthorizedError("User not authenticated"))
		return
	}

	// Fetch existing restaurant to check ownership
	existing, err := services.GetRestaurantByID(request.Context(), id)
	if err != nil {
		errors.ErrorResponse(writer, request, err)
		return
	}

	// Authorization and Input Sanitization
	switch user.Role {
	case types.RoleManagement:
		// 1. Ownership Check
		if existing.UserID == nil || *existing.UserID != user.UserID {
			errors.ErrorResponse(writer, request, errors.ForbiddenError("You do not have permission to update this restaurant"))
			return
		}
		// 2. Vulnerability Check: Mass Assignment Prevention
		// Management CANNOT update: Status, UserID (Transfer), Rating
		input.Status = ""
		input.UserID = nil
		input.Rating = nil
		
	case types.RoleAdmin:
		// Admin can update everything, but typically shouldn't manually update Rating via this endpoint 
		// unless correcting data. We will allow it for Admin flexibility.
	default:
		errors.ErrorResponse(writer, request, errors.ForbiddenError("You do not have permission"))
		return
	}

	resp, err := services.UpdateRestaurant(request.Context(), id, input)
	if err != nil {
		errors.ErrorResponse(writer, request, err)
		return
	}

	utils.WriteJSON(writer, http.StatusOK, map[string]interface{}{
		"title": "Updated Restaurant Successfully",
		"data":  resp,
	})
}
