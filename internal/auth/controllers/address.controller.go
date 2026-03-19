package controllers

import (
	"net/http"

	"github.com/alibaba0010/postgres-api/internal/auth/repositories"
	"github.com/alibaba0010/postgres-api/internal/common/errors"
	"github.com/alibaba0010/postgres-api/internal/common/guards"
	"github.com/alibaba0010/postgres-api/internal/utils"
	commondto "github.com/alibaba0010/postgres-api/internal/common/dto"

	"github.com/gorilla/mux"
)

// DeleteUserAddressHandler deletes one of the authenticated user's addresses.
// DELETE /user/addresses/{addressId}
func DeleteUserAddressHandler(writer http.ResponseWriter, request *http.Request) {
	authenticatedUser := guards.ExtractAuthenticatedUser(request)
	if authenticatedUser == nil {
		errors.ErrorResponse(writer, request, errors.UnauthorizedError("user not authenticated"))
		return
	}

	vars := mux.Vars(request)
	addressID := vars["addressId"]
	if addressID == "" {
		errors.ErrorResponse(writer, request, errors.ValidationError("addressId is required"))
		return
	}

	if err := repositories.UserRepo.DeleteAddress(request.Context(), authenticatedUser.UserID, addressID); err != nil {
		errors.ErrorResponse(writer, request, errors.NotFoundError("address not found or does not belong to user"))
		return
	}

	utils.WriteJSON(writer, http.StatusOK, commondto.MessageResponse{
		Title:   "Address deleted",
		Message: "Address removed successfully",
	})
}

// SetDefaultUserAddressHandler marks an address as the default for the authenticated user.
// PATCH /user/addresses/{addressId}/default
func SetDefaultUserAddressHandler(writer http.ResponseWriter, request *http.Request) {
	authenticatedUser := guards.ExtractAuthenticatedUser(request)
	if authenticatedUser == nil {
		errors.ErrorResponse(writer, request, errors.UnauthorizedError("user not authenticated"))
		return
	}

	vars := mux.Vars(request)
	addressID := vars["addressId"]
	if addressID == "" {
		errors.ErrorResponse(writer, request, errors.ValidationError("addressId is required"))
		return
	}

	if err := repositories.UserRepo.SetDefaultAddress(request.Context(), nil, authenticatedUser.UserID, addressID); err != nil {
		errors.ErrorResponse(writer, request, errors.NotFoundError("address not found or does not belong to user"))
		return
	}

	utils.WriteJSON(writer, http.StatusOK, commondto.MessageResponse{
		Title:   "Default address updated",
		Message: "Default address set successfully",
	})
}
