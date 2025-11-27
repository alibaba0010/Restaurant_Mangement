package controllers

import (
	"encoding/json"
	"net/http"

	"github.com/alibaba0010/postgres-api/internal/errors"
	"github.com/alibaba0010/postgres-api/internal/guards"
	"github.com/alibaba0010/postgres-api/internal/services"
)

// CurrentUserHandler retrieves and returns the authenticated user's information
// @Summary Get current authenticated user
// @Description Retrieve information about the currently authenticated user
// @Tags users
// @Security Bearer
// @Produce json
// @Success 200 {object} services.CurrentUserResponse "Successfully retrieved current user"
// @Failure 401 {object} errors.ErrorResponseStruct "Unauthorized - no valid token"
// @Failure 404 {object} errors.ErrorResponseStruct "User not found"
// @Failure 500 {object} errors.ErrorResponseStruct "Internal server error"
// @Router /user [get]
func CurrentUserHandler(writer http.ResponseWriter, request *http.Request) {
	// Extract authenticated user from request headers (set by AuthMiddleware)
	authenticatedUser := guards.ExtractAuthenticatedUser(request)
	if authenticatedUser == nil {
		errors.ErrorResponse(writer, request, errors.UnauthorizedError("user not authenticated"))
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
