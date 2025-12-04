package controllers

import (
	"encoding/json"
	"net/http"

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

