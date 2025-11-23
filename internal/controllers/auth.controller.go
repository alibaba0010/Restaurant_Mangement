package controllers

import (
	"encoding/json"
	"net/http"

	"github.com/alibaba0010/postgres-api/internal/dto"
	"github.com/alibaba0010/postgres-api/internal/errors"
	"github.com/alibaba0010/postgres-api/internal/services"
)

// SignupHandler godoc
//
//	@Summary		User Signup
//	@Description	Creates a new user account
//	@Tags			Auth
//	@Accept			json
//	@Produce		json
//	@Param			body	body		dto.SignupInput	true	"Signup request"
//	@Success		201	{object}	map[string]interface{} "User created successfully"
//	@Failure		400	{object}	map[string]string		"Validation error"
//	@Failure		409	{object}	map[string]string		"Duplicate email"
//	@Failure		500	{object}	map[string]string		"Internal server error"
//	@Router			/auth/signup [post]
func SignupHandler(writer http.ResponseWriter, request *http.Request) {
	var input dto.SignupInput

	// Decode JSON
	if err := json.NewDecoder(request.Body).Decode(&input); err != nil {
		errors.ErrorResponse(writer, request, errors.ValidationError("Invalid JSON body"))
		return
	}

	user, appErr := services.RegisterUser(request.Context(), input)
	if appErr != nil {
		errors.ErrorResponse(writer, request, appErr)
		return
	}

	// Return success response (without password) using DTO types
	resp := dto.SignUpResponse{
		Title: "User created successfully",
		Data: dto.SignUpData{
			ID:    user.ID,
			Name:  user.Name,
			Email: user.Email,
		},
	}

	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(writer).Encode(resp)
}
func VerifyEmailHandler(writer http.ResponseWriter, request *http.Request) {
	// Implementation for email verification goes here
}