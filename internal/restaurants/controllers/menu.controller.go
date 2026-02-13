package controllers

import (
	"encoding/json"
	"net/http"
	"strings"

	commondto "github.com/alibaba0010/postgres-api/internal/common/dto"
	"github.com/alibaba0010/postgres-api/internal/common/errors"
	"github.com/alibaba0010/postgres-api/internal/common/guards"
	"github.com/alibaba0010/postgres-api/internal/restaurants/dto"
	"github.com/alibaba0010/postgres-api/internal/restaurants/services"
	"github.com/alibaba0010/postgres-api/internal/utils"
	"github.com/gorilla/mux"
)

// MenuController holds the menu service and provides HTTP handlers
type MenuController struct {
	service *services.MenuService
}

// NewMenuController creates a new menu controller with the given service
func NewMenuController(menuService *services.MenuService) *MenuController {
	return &MenuController{
		service: menuService,
	}
}

type MenuControllerInterface interface {
	InitiateMultipartUploadHandler(writer http.ResponseWriter, request *http.Request)
	GetMultipartPartURLHandler(writer http.ResponseWriter, request *http.Request)
	CompleteMultipartUploadHandler(writer http.ResponseWriter, request *http.Request)
	GetMenuUploadURLHandler(writer http.ResponseWriter, request *http.Request)
	UploadMenuMediaHandler(writer http.ResponseWriter, request *http.Request)
	CreateMenuHandler(writer http.ResponseWriter, request *http.Request)
	GetMenuMediaHandler(writer http.ResponseWriter, request *http.Request)
	ListMenuHandler(writer http.ResponseWriter, request *http.Request)
	DeleteMenuMediaHandler(writer http.ResponseWriter, request *http.Request)
}

// InitiateMultipartUploadHandler handles the initiation of a multipart upload, returns upload id and key
func (mc *MenuController) InitiateMultipartUploadHandler(writer http.ResponseWriter, request *http.Request) {
	var input dto.InitiateMultipartUploadInput

	if err := json.NewDecoder(request.Body).Decode(&input); err != nil {
		errors.ErrorResponse(writer, request, errors.ValidationError("Invalid request body"))
		return
	}

	user := guards.ExtractAuthenticatedUser(request)
	if user == nil {
		errors.ErrorResponse(writer, request, errors.UnauthorizedError("Authentication required"))
		return
	}

	resp, err := mc.service.InitiateMultipartUpload(request.Context(), user.UserID, input.Filename, input.ContentType)
	if err != nil {
		errors.ErrorResponse(writer, request, errors.InternalError(err))
		return
	}

	utils.WriteJSON(writer, http.StatusCreated, dto.InitiateMultipartUploadResponseHandler{
		Title: "Multipart upload initiated",
		Data:  *resp,
	})
}

// GetMultipartPartURLHandler handles generating a presigned URL for a part
func (mc *MenuController) GetMultipartPartURLHandler(writer http.ResponseWriter, request *http.Request) {
	query := request.URL.Query()
	key := query.Get("key")
	uploadID := query.Get("upload_id")
	partNumberStr := query.Get("part_number")

	if key == "" || uploadID == "" || partNumberStr == "" {
		errors.ErrorResponse(writer, request, errors.ValidationError("key, upload_id, and part_number are required"))
		return
	}

	partNumber := utils.ParseInt(partNumberStr, 0)
	if partNumber <= 0 {
		errors.ErrorResponse(writer, request, errors.ValidationError("part_number must be a positive integer"))
		return
	}

	url, err := mc.service.GetPartPresignedURL(request.Context(), key, uploadID, int32(partNumber))
	if err != nil {
		errors.ErrorResponse(writer, request, errors.InternalError(err))
		return
	}

	utils.WriteJSON(writer, http.StatusOK, dto.GenerateMultipartPartURLResponse{
		Title: "Presigned part URL generated",
		Data:  url,
	})
}

// CompleteMultipartUploadHandler handles the completion of a multipart upload
func (mc *MenuController) CompleteMultipartUploadHandler(writer http.ResponseWriter, request *http.Request) {
	var input dto.CompleteMultipartUploadInput
	if err := json.NewDecoder(request.Body).Decode(&input); err != nil {
		errors.ErrorResponse(writer, request, errors.ValidationError("Invalid request body"))
		return
	}

	publicURL, err := mc.service.CompleteMultipartUpload(request.Context(), input.Key, input.UploadID, input.Parts)
	if err != nil {
		errors.ErrorResponse(writer, request, errors.InternalError(err))
		return
	}

	utils.WriteJSON(writer, http.StatusOK, dto.CompleteMultipartUploadResponse{
		Title: "Multipart upload completed",
		Data:  dto.SingleURLResponse{URL: publicURL},
	})
}

// GetMenuUploadURLHandler handles the request for a presigned URL for menu media uploads
func (mc *MenuController) GetMenuUploadURLHandler(writer http.ResponseWriter, request *http.Request) {
	filename := request.URL.Query().Get("filename")
	contentType := request.URL.Query().Get("content_type")

	if filename == "" || contentType == "" {
		errors.ErrorResponse(writer, request, errors.ValidationError("filename and content_type are required"))
		return
	}

	user := guards.ExtractAuthenticatedUser(request)
	if user == nil {
		errors.ErrorResponse(writer, request, errors.UnauthorizedError("Authentication required"))
		return
	}



	uploadURL, publicURL, appErr := mc.service.GetUploadURL(request.Context(), user.UserID, filename, contentType)
	if appErr != nil {
		errors.ErrorResponse(writer, request, appErr)
		return
	}

	utils.WriteJSON(writer, http.StatusOK, dto.GetMenuUploadURLResponse{
		Title: "Presigned URL generated successfully",
		Data: dto.URLResponse{
			UploadURL: uploadURL,
			PublicURL: publicURL,
		},
	})
}

// UploadMenuMediaHandler handles direct media upload (Multipart Form)
func (mc *MenuController) UploadMenuMediaHandler(writer http.ResponseWriter, request *http.Request) {
	// Parse the multipart form (max 50MB)
	err := request.ParseMultipartForm(50 << 20)
	if err != nil {
		errors.ErrorResponse(writer, request, errors.ValidationError("File too large or invalid form"))
		return
	}

	file, header, err := request.FormFile("file")
	if err != nil {
		errors.ErrorResponse(writer, request, errors.ValidationError("File is required"))
		return
	}
	defer file.Close()

	user := guards.ExtractAuthenticatedUser(request)
	if user == nil {
		errors.ErrorResponse(writer, request, errors.UnauthorizedError("Authentication required"))
		return
	}

	contentType := header.Header.Get("Content-Type")
	filename := header.Filename

	publicURL, appErr := mc.service.UploadMedia(request.Context(), user.UserID, filename, contentType, file)
	if appErr != nil {
		errors.ErrorResponse(writer, request, appErr)
		return
	}

	utils.WriteJSON(writer, http.StatusOK, dto.UploadMenuMediaResponse{
		Title: "Upload successful",
		Data:  dto.SingleURLResponse{URL: publicURL},
	})
}

// CreateMenuHandler handles the creation of a menu item
func (mc *MenuController) CreateMenuHandler(writer http.ResponseWriter, request *http.Request) {
	var input dto.CreateMenuInput
	if err := json.NewDecoder(request.Body).Decode(&input); err != nil {
		errors.ErrorResponse(writer, request, errors.ValidationError("Invalid request body"))
		return
	}

	// Validate input
	if err := utils.ValidateInput(input); err != nil {
		errors.ErrorResponse(writer, request, err)
		return
	}

	user := guards.ExtractAuthenticatedUser(request)
	if user == nil {
		errors.ErrorResponse(writer, request, errors.UnauthorizedError("Authentication required"))
		return
	}

	menu, appErr := mc.service.Create(request.Context(), input, user.UserID, user.Role)
	if appErr != nil {
		errors.ErrorResponse(writer, request, appErr)
		return
	}

	utils.WriteJSON(writer, http.StatusCreated, commondto.SingleDataResponse[*dto.MenuResponse]{
		Title: "Menu item created successfully",
		Data:  menu,
	})
}

// GetMenuHandler handles retrieving a single menu item
func (mc *MenuController) GetMenuHandler(writer http.ResponseWriter, request *http.Request) {
	vars := mux.Vars(request)
	id := vars["id"]

	menu, appErr := mc.service.GetByID(request.Context(), id)
	if appErr != nil {
		errors.ErrorResponse(writer, request, appErr)
		return
	}

	utils.WriteJSON(writer, http.StatusOK, dto.GetMenuByIDResponse{
		Title:    "Menu details",
		Response: *mc.service.MapToResponse(menu),
	})
}

// ListMenusHandler handles listing and filtering menu items
func (mc *MenuController) ListMenusHandler(writer http.ResponseWriter, request *http.Request) {
	params := utils.ParseListParams(request)
	query := request.URL.Query()
	restaurantID := query.Get("restaurant_id")
	categoryID := query.Get("category_id")
	tagsStr := query.Get("tags")
	var tags []string
	if tagsStr != "" {
		tags = strings.Split(tagsStr, ",")
	}

	var minPrice, maxPrice *float64
	if mp := query.Get("min_price"); mp != "" {
		p := utils.ParseFloat(mp, 0)
		minPrice = &p
	}
	if mp := query.Get("max_price"); mp != "" {
		p := utils.ParseFloat(mp, 0)
		maxPrice = &p
	}

	var isAvailable *bool
	if ia := query.Get("is_available"); ia != "" {
		b := ia == "true"
		isAvailable = &b
	}

	menus, nextCursor, hasMore, total, appErr := mc.service.ListMenus(request.Context(), params.Limit, params.Cursor, params.Query, restaurantID, categoryID, tags, minPrice, maxPrice, isAvailable, params.SortBy, params.Order)
	if appErr != nil {
		errors.ErrorResponse(writer, request, appErr)
		return
	}

	utils.WriteJSON(writer, http.StatusOK, dto.MenusListResponse{
		Data: menus,
		Meta: commondto.CursorMeta{
			NextCursor: nextCursor,
			HasMore:    hasMore,
			Total:      total,
		},
	})
}

// UpdateMenuHandler handles updating an existing menu item
func (mc *MenuController) UpdateMenuHandler(writer http.ResponseWriter, request *http.Request) {
	vars := mux.Vars(request)
	id := vars["id"]

	var input dto.UpdateMenuInput
	if err := json.NewDecoder(request.Body).Decode(&input); err != nil {
		errors.ErrorResponse(writer, request, errors.ValidationError("Invalid request body"))
		return
	}

	// Validate input
	if err := utils.ValidateInput(input); err != nil {
		errors.ErrorResponse(writer, request, err)
		return
	}

	user := guards.ExtractAuthenticatedUser(request)
	if user == nil {
		errors.ErrorResponse(writer, request, errors.UnauthorizedError("Authentication required"))
		return
	}

	updatedMenu, appErr := mc.service.Update(request.Context(), id, input, user.UserID, user.Role)
	if appErr != nil {
		errors.ErrorResponse(writer, request, appErr)
		return
	}

	utils.WriteJSON(writer, http.StatusOK, commondto.SingleDataResponse[*dto.MenuResponse]{
		Title: "Menu item updated successfully",
		Data:  updatedMenu,
	})
}
