package controllers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/alibaba0010/postgres-api/internal/dto"
	"github.com/alibaba0010/postgres-api/internal/errors"
	"github.com/alibaba0010/postgres-api/internal/guards"
	"github.com/alibaba0010/postgres-api/internal/services"
	"github.com/alibaba0010/postgres-api/internal/utils"

	"github.com/gorilla/mux"
)

// CurrentUserHandler returns the authenticated user's profile
func CurrentUserHandler(writer http.ResponseWriter, request *http.Request) {
	authenticatedUser := guards.ExtractAuthenticatedUser(request)
	if authenticatedUser == nil {
		errors.ErrorResponse(writer, request, errors.UnauthorizedError("user not authenticated"))
		return
	}

	user, appErr := services.GetCurrentUserByID(request.Context(), authenticatedUser.UserID)
	if appErr != nil {
		errors.ErrorResponse(writer, request, appErr)
		return
	}

	utils.WriteJSON(writer, http.StatusOK, user)
}

// GetUserByIDHandler retrieves a user by ID (admin only via router middleware)
func GetUserByIDHandler(writer http.ResponseWriter, request *http.Request) {
	vars := mux.Vars(request)
	userID := vars["id"]

	if userID == "" {
		errors.ErrorResponse(writer, request, errors.ValidationError("user id is required"))
		return
	}

	user, appErr := services.GetUserByIDPublic(request.Context(), userID)
	if appErr != nil {
		errors.ErrorResponse(writer, request, appErr)
		return
	}

	utils.WriteJSON(writer, http.StatusOK, user)
}

// UpdateUserHandler updates the authenticated user's information
func UpdateUserHandler(writer http.ResponseWriter, request *http.Request) {
	authenticatedUser := guards.ExtractAuthenticatedUser(request)
	if authenticatedUser == nil {
		errors.ErrorResponse(writer, request, errors.UnauthorizedError("user not authenticated"))
		return
	}

	var input dto.UpdateAddressInput
	if err := json.NewDecoder(request.Body).Decode(&input); err != nil {
		errors.ErrorResponse(writer, request, errors.ValidationError("invalid request body"))
		return
	}

	response, appErr := services.UpdateUser(request.Context(), authenticatedUser.UserID, input)
	if appErr != nil {
		errors.ErrorResponse(writer, request, appErr)
		return
	}

	utils.WriteJSON(writer, http.StatusOK, response)

}

// GetAllUsersHandler returns a paginated list of users (admin only)
func GetAllUsersHandler(writer http.ResponseWriter, request *http.Request) {
	// parse query params
	q := request.URL.Query()

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
	role := q.Get("role")
	sortBy := q.Get("sort_by")
	order := q.Get("order")

	users, total, appErr := services.GetAllUsers(request.Context(), page, pageSize, search, role, sortBy, order)
	if appErr != nil {
		errors.ErrorResponse(writer, request, appErr)
		return
	}

	totalPages := 0
	if pageSize > 0 {
		totalPages = int((total + int64(pageSize) - 1) / int64(pageSize))
	}

	resp := dto.UsersListResponse{
		Title: "Success",
		Data:  users,
		Meta: dto.PaginationMeta{
			Page:       page,
			PageSize:   pageSize,
			Total:      total,
			TotalPages: totalPages,
		},
	}

	utils.WriteJSON(writer, http.StatusOK, resp)
}

