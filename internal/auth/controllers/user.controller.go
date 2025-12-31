package controllers

import (
	"encoding/json"
	"net/http"

	"github.com/alibaba0010/postgres-api/internal/auth/dto"
	"github.com/alibaba0010/postgres-api/internal/auth/services"
	commondto "github.com/alibaba0010/postgres-api/internal/common/dto"
	"github.com/alibaba0010/postgres-api/internal/common/errors"
	"github.com/alibaba0010/postgres-api/internal/common/guards"
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

	var input dto.UpdateUserInput
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
	params := utils.ParseListParams(request)
	role := request.URL.Query().Get("role")

	users, total, appErr := services.GetAllUsers(request.Context(), params.Page, params.PageSize, params.Query, role, params.SortBy, params.Order)
	if appErr != nil {
		errors.ErrorResponse(writer, request, appErr)
		return
	}

	resp := dto.UsersListResponse{
		Title: "Users retrieved successfully",
		Data:  users,
		Meta: commondto.PaginationMeta{
			Page:       params.Page,
			PageSize:   params.PageSize,
			Total:      total,
			TotalPages: utils.CalculateTotalPages(total, params.PageSize),
		},
	}

	utils.WriteJSON(writer, http.StatusOK, resp)
}

// UpdateUserRoleHandler updates a user's role (admin only)
func UpdateUserRoleStatusHandler(writer http.ResponseWriter, request *http.Request) {
	vars := mux.Vars(request)
	userID := vars["id"]

	var input dto.UpdateUserRoleInput
	if err := json.NewDecoder(request.Body).Decode(&input); err != nil {
		errors.ErrorResponse(writer, request, errors.ValidationError("invalid request body"))
		return
	}

	response, appErr := services.UpdateUserRoleStatus(request.Context(), userID, input)
	if appErr != nil {
		errors.ErrorResponse(writer, request, appErr)
		return
	}

	utils.WriteJSON(writer, http.StatusOK, response)
}
