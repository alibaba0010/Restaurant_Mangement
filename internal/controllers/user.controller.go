package controllers

import (
	"encoding/json"
	"net/http"

	"github.com/alibaba0010/postgres-api/internal/dto"
	"github.com/alibaba0010/postgres-api/internal/errors"
	"github.com/alibaba0010/postgres-api/internal/guards"
	"github.com/alibaba0010/postgres-api/internal/services"
	"github.com/gorilla/mux"
)

func CurrentUserHandler(writer http.ResponseWriter, request *http.Request) {
	// Extract authenticated user from request headers (set by AuthMiddleware)
	authenticatedUser := guards.ExtractAuthenticatedUser(request)
	if authenticatedUser == nil {
		errors.ErrorResponse(writer, request, errors.UnauthorizedError("User not authenticated"))
		return
	}

	// Fetch user from database
	user, appErr := services.GetCurrentUserByID(request.Context(), authenticatedUser.UserID)
	if appErr != nil {
		errors.ErrorResponse(writer, request, appErr)
		return
	}

	// Write response
	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(http.StatusOK)
	json.NewEncoder(writer).Encode(user)
}

// GetUserByIDHandler retrieves a specific user by ID (public endpoint with role checks)
// Only accessible to authenticated admin
func GetUserByIDHandler(writer http.ResponseWriter, request *http.Request) {
	vars := mux.Vars(request)
	userID := vars["id"]

	if userID == "" {
		errors.ErrorResponse(writer, request, errors.ValidationError("user id is required"))
		return
	}

	// Check if requesting own profile or has admin/management role
	authenticatedUser := guards.ExtractAuthenticatedUser(request)
	isOwner := authenticatedUser != nil && authenticatedUser.UserID == userID
	isAdmin := authenticatedUser != nil && authenticatedUser.Role == "admin"
	isManagement := authenticatedUser != nil && authenticatedUser.Role == "management"

	if !isOwner && !isAdmin && !isManagement {
		errors.ErrorResponse(writer, request, errors.ForbiddenError("you don't have permission to view this user"))
		return
	}

	// Fetch user
	user, appErr := services.GetUserByIDPublic(request.Context(), userID)
	if appErr != nil {
		errors.ErrorResponse(writer, request, appErr)
		return
	}

	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(http.StatusOK)
	json.NewEncoder(writer).Encode(user)
}

// UpdateUserAddressHandler updates the current user's address
func UpdateUserAddressHandler(writer http.ResponseWriter, request *http.Request) {
	authenticatedUser := guards.ExtractAuthenticatedUser(request)
	if authenticatedUser == nil {
		errors.ErrorResponse(writer, request, errors.UnauthorizedError("user not authenticated"))
		return
	}

	// Parse request body
	var input dto.UpdateAddressInput
	if err := json.NewDecoder(request.Body).Decode(&input); err != nil {
		errors.ErrorResponse(writer, request, errors.ValidationError("invalid request body"))
		return
	}

	// Call service to update address
	response, appErr := services.UpdateUserAddress(request.Context(), authenticatedUser.UserID, input)
	if appErr != nil {
		errors.ErrorResponse(writer, request, appErr)
		return
	}

	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(http.StatusOK)
	json.NewEncoder(writer).Encode(response)
}

