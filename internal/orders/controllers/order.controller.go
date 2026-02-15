package controllers

import (
	"encoding/json"
	"net/http"

	commonDto "github.com/alibaba0010/postgres-api/internal/common/dto"
	"github.com/alibaba0010/postgres-api/internal/common/errors"
	"github.com/alibaba0010/postgres-api/internal/common/guards"
	"github.com/alibaba0010/postgres-api/internal/orders/dto"
	"github.com/alibaba0010/postgres-api/internal/orders/services"

	"github.com/alibaba0010/postgres-api/internal/utils"
	"github.com/gorilla/mux"
)

type OrderController struct {
	service *services.OrderService
}
type OrderControllerInterface interface { 
	CreateOrderHandler(writer http.ResponseWriter, request *http.Request)
	GetUserOrdersHandler(writer http.ResponseWriter, request *http.Request)
	GetOrderByIDHandler(writer http.ResponseWriter, request *http.Request)
	UpdateOrderStatusHandler(writer http.ResponseWriter, request *http.Request)
}
// NewOrderController creates a new instance of OrderController.
func NewOrderController(service *services.OrderService) *OrderController {
	return &OrderController{service: service}
}

// CreateOrderHandler handles the HTTP request for creating a new order.
func (oc *OrderController) CreateOrderHandler(writer http.ResponseWriter, request *http.Request) {
	var input dto.CreateOrderInput
	if err := json.NewDecoder(request.Body).Decode(&input); err != nil {
		errors.ErrorResponse(writer, request, errors.ValidationError("Invalid request body"))
		return
	}

	user := guards.ExtractAuthenticatedUser(request)
	if user == nil {
		errors.ErrorResponse(writer, request, errors.UnauthorizedError("Authentication required"))
		return
	}

	resp, appErr := oc.service.CreateOrder(request.Context(), user.UserID, input)
	if appErr != nil {
		errors.ErrorResponse(writer, request, appErr)
		return
	}

	utils.WriteJSON(writer, http.StatusCreated, resp)
}

// GetUserOrdersHandler handles the HTTP request for listing a user's orders.
func (oc *OrderController) GetUserOrdersHandler(writer http.ResponseWriter, request *http.Request) {
	user := guards.ExtractAuthenticatedUser(request)
	if user == nil {
		errors.ErrorResponse(writer, request, errors.UnauthorizedError("Authentication required"))
		return
	}

	params := utils.ParseListParams(request)

	orders, nextCursor, hasMore, appErr := oc.service.GetUserOrders(request.Context(), user.UserID, params.Limit, params.Cursor)
	if appErr != nil {
		errors.ErrorResponse(writer, request, appErr)
		return
	}

	utils.WriteJSON(writer, http.StatusOK, dto.UserOrdersResponse{
		Orders:     orders,
		NextCursor: nextCursor,
		HasMore:    hasMore,
	})
}

// GetOrderByIDHandler handles the HTTP request for retrieving a single order by ID.
func (oc *OrderController) GetOrderByIDHandler(writer http.ResponseWriter, request *http.Request) {
	vars := mux.Vars(request)
	id := vars["id"]

	user := guards.ExtractAuthenticatedUser(request)
	if user == nil {
		errors.ErrorResponse(writer, request, errors.UnauthorizedError("Authentication required"))
		return
	}

	order, appErr := oc.service.GetOrderByID(request.Context(), id, *user)
	if appErr != nil {
		errors.ErrorResponse(writer, request, appErr)
		return
	}

	utils.WriteJSON(writer, http.StatusOK, order)
}

// UpdateOrderStatusHandler handles the HTTP request for updating an order's status.
func (oc *OrderController) UpdateOrderStatusHandler(writer http.ResponseWriter, request *http.Request) {
	vars := mux.Vars(request)
	id := vars["id"]

	var input dto.UpdateOrderStatusInput
	if err := json.NewDecoder(request.Body).Decode(&input); err != nil {
		errors.ErrorResponse(writer, request, errors.ValidationError("Invalid request body"))
		return
	}

	user := guards.ExtractAuthenticatedUser(request)
	if user == nil {
		errors.ErrorResponse(writer, request, errors.UnauthorizedError("Authentication required"))
		return
	}

	appErr := oc.service.UpdateOrderStatus(request.Context(), id, input.Status, *user)
	if appErr != nil {
		errors.ErrorResponse(writer, request, appErr)
		return
	}

	utils.WriteJSON(writer, http.StatusOK, commonDto.MessageResponse{
		Title:   "Order Updated",
		Message: "Order status updated successfully",
	})
}
