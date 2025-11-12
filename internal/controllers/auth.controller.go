package controllers


import (
	"encoding/json"
	"net/http"

	"golang.org/x/crypto/bcrypt"
	"github.com/go-playground/validator/v10"
	"github.com/alibaba0010/postgres-api/internal/database"
	"github.com/alibaba0010/postgres-api/internal/errors"
	"github.com/alibaba0010/postgres-api/internal/dto"
	"github.com/alibaba0010/postgres-api/internal/models"


	"go.uber.org/zap"
)

var validate = validator.New()

func SignupHandler(w http.ResponseWriter, r *http.Request, logger *zap.Logger) {
	var input dto.SignupInput

	// Decode JSON
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		errors.Respond(w, logger, errors.ValidationError("Invalid JSON body"))
		return
	}

	// Validate input
	if err := validate.Struct(input); err != nil {
		for _, e := range err.(validator.ValidationErrors) {
			msg := e.Field() + " failed validation: " + e.Tag()
			errors.Respond(w, logger, errors.ValidationError(msg))
			return
		}
	}

	// Check if user already exists
	exists, err := database.DB.NewSelect().Model((*models.User)(nil)).
		Where("email = ?", input.Email).
		Exists(r.Context())
	if err != nil {
		errors.Respond(w, logger, errors.InternalError(err))
		return
	}
	if exists {
		errors.Respond(w, logger, errors.DuplicateError("email"))
		return
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		errors.Respond(w, logger, errors.InternalError(err))
		return
	}

	// Save new user
	user := &models.User{
		Name:     input.Name,
		Email:    input.Email,
		Password: string(hashedPassword),
	}
	_, err = database.DB.NewInsert().Model(user).Exec(r.Context())
	if err != nil {
		errors.Respond(w, logger, errors.InternalError(err))
		return
	}

	// Return success response (without password)
	response := map[string]interface{}{
		"title":   "Success",
		"message": "User created successfully",
		"user": map[string]interface{}{
			"id":    user.ID,
			"name":  user.Name,
			"email": user.Email,
		},
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}
