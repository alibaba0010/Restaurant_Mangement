package controllers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/alibaba0010/postgres-api/internal/common/errors"
	"github.com/alibaba0010/postgres-api/internal/common/guards"
	"github.com/alibaba0010/postgres-api/internal/common/types"
	"github.com/alibaba0010/postgres-api/internal/restaurants/dto"
	"github.com/alibaba0010/postgres-api/internal/restaurants/services"
	"github.com/alibaba0010/postgres-api/internal/utils"
	"github.com/gorilla/mux"
)

// CreateRestaurantHandler handles the creation of a new restaurant
func CreateRestaurantHandler(w http.ResponseWriter, r *http.Request) {
	var input dto.CreateRestaurantInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		errors.ErrorResponse(w, r, errors.ValidationError("Invalid request body"))
		return
	}

	// Set the current user as the owner
	if user := guards.ExtractAuthenticatedUser(r); user != nil {
		input.UserID = &user.UserID
	}

	resp, err := services.CreateRestaurant(r.Context(), input)
	if err != nil {
		errors.ErrorResponse(w, r, err)
		return
	}

	utils.WriteJSON(w, http.StatusCreated, map[string]interface{}{
		"title": "Success",
		"data":  resp,
	})
}

// GetRestaurantHandler retrieves a restaurant by ID
func GetRestaurantHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	resp, err := services.GetRestaurantByID(r.Context(), id)
	if err != nil {
		errors.ErrorResponse(w, r, err)
		return
	}

	// Check ownership if management role
	if user := guards.ExtractAuthenticatedUser(r); user != nil {
		if user.Role == types.RoleManagement {
			if resp.UserID == nil {
				errors.ErrorResponse(w, r, errors.ForbiddenError("You do not have permission to view this restaurant"))
				return
			}
			// Compare UUIDs
			if *resp.UserID != user.UserID {
				errors.ErrorResponse(w, r, errors.ForbiddenError("You do not have permission to view this restaurant"))
				return
			}
		}
	}

	utils.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"title": "Success",
		"data":  resp,
	})
}

// ListRestaurantsHandler lists restaurants with pagination
func ListRestaurantsHandler(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	page := 1
	if p := q.Get("page"); p != "" {
		if v, err := strconv.Atoi(p); err == nil && v > 0 {
			page = v
		}
	}

	pageSize := 20
	if ps := q.Get("page_size"); ps != "" {
		if v, err := strconv.Atoi(ps); err == nil && v > 0 {
			pageSize = v
		}
	}

	search := q.Get("q")
	
	// Filter logic based on user role
	var filterUserID *string
	if user := guards.ExtractAuthenticatedUser(r); user != nil {
		switch user.Role {
		case types.RoleManagement:
			// Management -> Only own restaurants
			filterUserID = &user.UserID
		case types.RoleAdmin:
			// Admin -> All restaurants (no userID filter)
		default:
			errors.ErrorResponse(w, r, errors.UnauthorizedError("Unauthorized access"))
			return
		}
	}

	data, total, err := services.GetAllRestaurants(r.Context(), page, pageSize, search, filterUserID)
	if err != nil {
		errors.ErrorResponse(w, r, err)
		return
	}

	totalPages := 0
	if pageSize > 0 {
		totalPages = int((total + int64(pageSize) - 1) / int64(pageSize))
	}

	meta := dto.PaginationMeta{
		Page:       page,
		PageSize:   pageSize,
		Total:      total,
		TotalPages: totalPages,
	}

	utils.WriteJSON(w, http.StatusOK, dto.RestaurantsListResponse{
		Title: "Success",
		Data:  data,
		Meta:  meta,
	})
}

// UpdateRestaurantHandler handles updating a restaurant
func UpdateRestaurantHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	var input dto.UpdateRestaurantInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		errors.ErrorResponse(w, r, errors.ValidationError("Invalid request body"))
		return
	}

	user := guards.ExtractAuthenticatedUser(r)
	if user == nil {
		errors.ErrorResponse(w, r, errors.UnauthorizedError("User not authenticated"))
		return
	}

	// Fetch existing restaurant to check ownership
	existing, err := services.GetRestaurantByID(r.Context(), id)
	if err != nil {
		errors.ErrorResponse(w, r, err)
		return
	}

	if user.Role == types.RoleManagement {
		// Check ownership
		if existing.UserID == nil || *existing.UserID != user.UserID {
			errors.ErrorResponse(w, r, errors.ForbiddenError("You do not have permission to update this restaurant"))
			return
		}
		// Disallow status updates for management
		// If input.Status is present and different from existing, blocking it or ignoring it?
		// User requirement: "management can only get restaurants created by him and update its name, description, address, capacity, delivery/takeaway available"
		// This implies they CANNOT update status.
		// We explicitly ignore/clear status if management
		input.Status = "" 
	} else if user.Role == types.RoleAdmin {
		// Admin can update status, so we leave it
	} else {
		// Other roles? Assume forbidden if not admin/management caught by middleware, 
		// but explicit check is safer
		errors.ErrorResponse(w, r, errors.ForbiddenError("You do not have permission"))
		return
	}

	resp, err := services.UpdateRestaurant(r.Context(), id, input)
	if err != nil {
		errors.ErrorResponse(w, r, err)
		return
	}

	utils.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"title": "Success",
		"data":  resp,
	})
}
