package controllers

import (
	"encoding/json"
	"net/http"

	"github.com/alibaba0010/postgres-api/internal/common/errors"
	"github.com/alibaba0010/postgres-api/internal/common/guards"
	"github.com/alibaba0010/postgres-api/internal/common/types"
	"github.com/alibaba0010/postgres-api/internal/restaurants/dto"
	"github.com/alibaba0010/postgres-api/internal/restaurants/services"
	"github.com/alibaba0010/postgres-api/internal/utils"
)

type CategoryController struct {
	categoryService *services.CategoryService
	menuService     *services.MenuService
}
type CategoryControllerInterface interface{
	CreateCategoryHandler(writer http.ResponseWriter, request http.Request)
	ListCategoriesHandler(writer http.ResponseWriter, request *http.Request)
}
func NewCategoryController(categoryService *services.CategoryService, menuService *services.MenuService) *CategoryController {
	return &CategoryController{
		categoryService: categoryService,
		menuService:     menuService,
	}
}

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

	// Permission: Only restaurant manager can create categories, Admin can do anything
	if user.Role == types.RoleManagement {
		if err := cc.menuService.AuthorizeRestaurantOwner(request.Context(), input.RestaurantID, user.UserID); err != nil {
			errors.ErrorResponse(writer, request, err)
			return
		}
	}

	resp, appErr := cc.categoryService.CreateCategory(request.Context(), input)
	if appErr != nil {
		errors.ErrorResponse(writer, request, appErr)
		return
	}

	utils.WriteJSON(writer, http.StatusCreated, resp)
}

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

