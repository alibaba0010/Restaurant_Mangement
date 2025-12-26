package controllers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/alibaba0010/postgres-api/internal/common/errors"
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

	data, total, err := services.GetAllRestaurants(r.Context(), page, pageSize, search)
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
