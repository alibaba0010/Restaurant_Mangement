package controllers

import (
	"encoding/json"
	"net/http"

	commondto "github.com/alibaba0010/postgres-api/internal/common/dto"
	"github.com/alibaba0010/postgres-api/internal/common/errors"
	"github.com/alibaba0010/postgres-api/internal/common/guards"
	"github.com/alibaba0010/postgres-api/internal/common/types"
	"github.com/alibaba0010/postgres-api/internal/restaurants/dto"
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
	if validationErrors := utils.ValidateStruct(input); validationErrors != nil {
		errors.ErrorResponse(writer, request, errors.ValidationError(validationErrors[0]))
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
	var filterUserID *string
	user := guards.ExtractAuthenticatedUser(request)

	if user != nil {
		switch user.Role {
		case types.RoleManagement:
			// Management -> Only own restaurants
			filterUserID = &user.UserID
		case types.RoleAdmin:
			// Admin -> All restaurants (Admin View)
			filterUserID = nil
		case types.RoleUser:
			// User -> All restaurants (Customer View)
			filterUserID = nil
		default:
			// Unknown role -> All restaurants (Public View fallback)
			filterUserID = nil
		}
	} else {
		// Unauthenticated -> All restaurants (Public View)
		filterUserID = nil
	}

	data, total, err := services.GetAllRestaurants(request.Context(), params.Page, params.PageSize, params.Query, filterUserID, params.SortBy, params.Order)
	if err != nil {
		errors.ErrorResponse(writer, request, err)
		return
	}

	utils.WriteJSON(writer, http.StatusOK, dto.RestaurantsListResponse{
		Title: "Restaurants retrieved successfully",
		Data:  data,
		Meta: commondto.PaginationMeta{
			Page:       params.Page,
			PageSize:   params.PageSize,
			Total:      total,
			TotalPages: utils.CalculateTotalPages(total, params.PageSize),
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

	switch user.Role {
	case types.RoleManagement:
		// Check ownership
		if existing.UserID == nil || *existing.UserID != user.UserID {
			errors.ErrorResponse(writer, request, errors.ForbiddenError("You do not have permission to update this restaurant"))
			return
		}
		// Disallow status updates for management
		// If input.Status is present and different from existing, blocking it or ignoring it?
		// User requirement: "management can only get restaurants created by him and update its name, description, address, capacity, delivery/takeaway available"
		// This implies they CANNOT update status.
		// We explicitly ignore/clear status if management
		input.Status = ""
	case types.RoleAdmin:
		// Admin can update status, so we leave it
	default:
		// Other roles? Assume forbidden if not admin/management caught by middleware,
		// but explicit check is safer
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
