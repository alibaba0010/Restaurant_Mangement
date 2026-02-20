package controllers

import (
	"encoding/json"
	"net/http"

	commondto "github.com/alibaba0010/postgres-api/internal/common/dto"
	"github.com/alibaba0010/postgres-api/internal/common/errors"
	"github.com/alibaba0010/postgres-api/internal/common/guards"
	"github.com/alibaba0010/postgres-api/internal/restaurants/dto"
	"github.com/alibaba0010/postgres-api/internal/restaurants/repositories"
	"github.com/alibaba0010/postgres-api/internal/restaurants/services"
	"github.com/alibaba0010/postgres-api/internal/utils"
	"github.com/gorilla/mux"
)

// CategoryController manages HTTP requests for menu categories,
// coordinating between the category service and restaurant ownership verification.
type CategoryController struct {
	categoryService *services.CategoryService
	menuService     *services.MenuService
	restaurantRepo  *repositories.RestaurantRepository
}

// CategoryControllerInterface defines the interface for category HTTP handlers.
type CategoryControllerInterface interface {
	// CreateCategoryHandler handles the request to create a new category.
	CreateCategoryHandler(writer http.ResponseWriter, request *http.Request)
	// ListCategoriesByRestaurantHandler handles GET /menu-categories?restaurant_id=...
	// to retrieve all	// ListCategoriesByRestaurantHandler handles GET /menu-categories?restaurant_id=...
	ListCategoriesByRestaurantHandler(writer http.ResponseWriter, request *http.Request)
	// UpdateCategoryHandler handles PUT /categories/{id}
	UpdateCategoryHandler(writer http.ResponseWriter, request *http.Request)
	// DeleteCategoryHandler handles DELETE /categories/{id}
	DeleteCategoryHandler(writer http.ResponseWriter, request *http.Request)
}

// NewCategoryController creates a new instance of CategoryController.
func NewCategoryController(categoryService *services.CategoryService, menuService *services.MenuService, restaurantRepo *repositories.RestaurantRepository) *CategoryController {
	return &CategoryController{
		categoryService: categoryService,
		menuService:     menuService,
		restaurantRepo:  restaurantRepo,
	}
}

// CreateCategoryHandler handles the HTTP request for creating a new menu category.
func (cc *CategoryController) CreateCategoryHandler(writer http.ResponseWriter, request *http.Request) {
	var input dto.CreateCategoryInput
	if err := json.NewDecoder(request.Body).Decode(&input); err != nil {
		errors.ErrorResponse(writer, request, errors.ValidationError("Invalid request body"))
		return
	}

	user := guards.ExtractAuthenticatedUser(request)
	if user == nil {
		errors.ErrorResponse(writer, request, errors.UnauthorizedError("Authentication required"))
		return
	}

	resp, appErr := cc.categoryService.CreateCategory(request.Context(), user, input)
	if appErr != nil {
		errors.ErrorResponse(writer, request, appErr)
		return
	}

	utils.WriteJSON(writer, http.StatusCreated, resp)
}


// ListCategoriesByRestaurantHandler handles the HTTP request for listing categories of a restaurant.
func (cc *CategoryController) ListCategoriesByRestaurantHandler(writer http.ResponseWriter, request *http.Request) {
	restaurantID := request.URL.Query().Get("restaurant_id")
	if restaurantID == "" {
		errors.ErrorResponse(writer, request, errors.ValidationError("restaurant_id is required"))
		return
	}

	categories, appErr := cc.categoryService.ListCategoriesByRestaurant(request.Context(), restaurantID)
	if appErr != nil {
		errors.ErrorResponse(writer, request, appErr)
		return
	}

	utils.WriteJSON(writer, http.StatusOK, categories)
}

// UpdateCategoryHandler handles category updates.
func (cc *CategoryController) UpdateCategoryHandler(writer http.ResponseWriter, request *http.Request) {
	vars := mux.Vars(request)
	id := vars["id"]

	var input dto.UpdateCategoryInput
	if err := json.NewDecoder(request.Body).Decode(&input); err != nil {
		errors.ErrorResponse(writer, request, errors.ValidationError("Invalid request body"))
		return
	}

	user := guards.ExtractAuthenticatedUser(request)
	if user == nil {
		errors.ErrorResponse(writer, request, errors.UnauthorizedError("Authentication required"))
		return
	}

	resp, appErr := cc.categoryService.UpdateCategory(request.Context(), user, id, input)
	if appErr != nil {
		errors.ErrorResponse(writer, request, appErr)
		return
	}

	utils.WriteJSON(writer, http.StatusOK, commondto.SingleDataResponse[*dto.CategoryResponse]{
		Title: "Category updated successfully",
		Data:  resp,
	})
}

// DeleteCategoryHandler handles category deletion.
func (cc *CategoryController) DeleteCategoryHandler(writer http.ResponseWriter, request *http.Request) {
	vars := mux.Vars(request)
	id := vars["id"]

	user := guards.ExtractAuthenticatedUser(request)
	if user == nil {
		errors.ErrorResponse(writer, request, errors.UnauthorizedError("Authentication required"))
		return
	}

	appErr := cc.categoryService.DeleteCategory(request.Context(), user, id)
	if appErr != nil {
		errors.ErrorResponse(writer, request, appErr)
		return
	}

	utils.WriteJSON(writer, http.StatusOK, commondto.MessageResponse{
		Title:   "Category deleted successfully",
		Message: "The category and its associations have been removed",
	})
}

