package controllers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	commondto "github.com/alibaba0010/postgres-api/internal/common/dto"
	"github.com/alibaba0010/postgres-api/internal/common/errors"
	"github.com/alibaba0010/postgres-api/internal/common/guards"
	"github.com/alibaba0010/postgres-api/internal/restaurants/dto"
	"github.com/alibaba0010/postgres-api/internal/restaurants/repositories"
	"github.com/alibaba0010/postgres-api/internal/restaurants/services"
	"github.com/alibaba0010/postgres-api/internal/utils"
	"github.com/gorilla/mux"
	"github.com/shopspring/decimal"
)

// MenuController holds the menu service and provides HTTP handlers
type MenuController struct {
	service        *services.MenuService
	restaurantRepo *repositories.RestaurantRepository
}

// NewMenuController creates a new menu controller with the given service
func NewMenuController(menuService *services.MenuService, restaurantRepo *repositories.RestaurantRepository) *MenuController {
	return &MenuController{
		service:        menuService,
		restaurantRepo: restaurantRepo,
	}
}

type MenuControllerInterface interface {
	InitiateMultipartUploadHandler(writer http.ResponseWriter, request *http.Request)
	GetMultipartPartURLHandler(writer http.ResponseWriter, request *http.Request)
	CompleteMultipartUploadHandler(writer http.ResponseWriter, request *http.Request)
	AbortMultipartUploadHandler(writer http.ResponseWriter, request *http.Request)
	GetMenuUploadURLHandler(writer http.ResponseWriter, request *http.Request)
	UploadMenuMediaHandler(writer http.ResponseWriter, request *http.Request)
	CreateMenuHandler(writer http.ResponseWriter, request *http.Request)
	GetMenuMediaHandler(writer http.ResponseWriter, request *http.Request)
	ListMenuHandler(writer http.ResponseWriter, request *http.Request)
	DeleteMenuHandler(writer http.ResponseWriter, request *http.Request)
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

// UploadMultipartPartHandler handles server-proxied chunk uploads to S3.
// The client sends the raw binary chunk body; the server forwards it to S3 using the
// AWS SDK (server-to-S3, no CORS preflight from the browser).
// Query params: key, upload_id, part_number
func (mc *MenuController) UploadMultipartPartHandler(writer http.ResponseWriter, request *http.Request) {
	query := request.URL.Query()
	key := query.Get("key")
	uploadID := query.Get("upload_id")
	partNumberStr := query.Get("part_number")

	if key == "" || uploadID == "" || partNumberStr == "" {
		errors.ErrorResponse(writer, request, errors.ValidationError("key, upload_id, and part_number are required"))
		return
	}

	partNumber, err := strconv.ParseInt(partNumberStr, 10, 32)
	if err != nil || partNumber <= 0 {
		errors.ErrorResponse(writer, request, errors.ValidationError("part_number must be a positive integer"))
		return
	}

	// Content-Length is required so S3 knows the part size
	contentLength := request.ContentLength
	if contentLength <= 0 {
		errors.ErrorResponse(writer, request, errors.ValidationError("Content-Length header is required"))
		return
	}

	etag, svcErr := mc.service.UploadMultipartPart(
		request.Context(),
		key,
		uploadID,
		int32(partNumber),
		request.Body,
		contentLength,
	)
	if svcErr != nil {
		errors.ErrorResponse(writer, request, errors.InternalError(svcErr))
		return
	}

	resp := dto.UploadMultipartPartResponse{
		Title: "Part uploaded successfully",
	}
	resp.Data.ETag = etag
	resp.Data.PartNumber = int32(partNumber)
	utils.WriteJSON(writer, http.StatusOK, resp)
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

// AbortMultipartUploadHandler handles the abortion of a multipart upload
func (mc *MenuController) AbortMultipartUploadHandler(writer http.ResponseWriter, request *http.Request) {
	var input dto.AbortMultipartUploadInput
	if err := json.NewDecoder(request.Body).Decode(&input); err != nil {
		errors.ErrorResponse(writer, request, errors.ValidationError("Invalid request body"))
		return
	}

	appErr := mc.service.AbortMultipartUpload(request.Context(), input.Key, input.UploadID)
	if appErr != nil {
		errors.ErrorResponse(writer, request, appErr)
		return
	}

	utils.WriteJSON(writer, http.StatusOK, dto.AbortMultipartUploadResponse{
		Title:   "Multipart upload aborted",
		Message: "The upload process was successfully canceled.",
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

	publicURL, appErr := mc.service.UploadMedia(request.Context(), user.UserID, filename, contentType, file, header.Size)
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

	menu, appErr := mc.service.CreateMenu(request.Context(), user, input)
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

	menu, appErr := mc.service.GetMenuByID(request.Context(), id)
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

	filter := dto.ListMenusFilter{
		RestaurantID: query.Get("restaurant_id"),
		CategoryID:   query.Get("category_id"),
		Limit:        params.Limit,
		Cursor:       params.Cursor,
		Query:        params.Query,
		SortBy:       params.SortBy,
		Order:        params.Order,
	}

	if tagsStr := query.Get("tags"); tagsStr != "" {
		filter.Tags = strings.Split(tagsStr, ",")
	}

	if mp := query.Get("min_price"); mp != "" {
		p := utils.ParseDecimal(mp, decimal.Zero)
		filter.MinPrice = &p
	}
	if mp := query.Get("max_price"); mp != "" {
		p := utils.ParseDecimal(mp, decimal.Zero)
		filter.MaxPrice = &p
	}

	if ia := query.Get("is_available"); ia != "" {
		b := ia == "true"
		filter.IsAvailable = &b
	}

	menus, nextCursor, hasMore, total, appErr := mc.service.ListMenus(request.Context(), filter)
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

	updatedMenu, appErr := mc.service.UpdateMenu(request.Context(), user, id, input)
	if appErr != nil {
		errors.ErrorResponse(writer, request, appErr)
		return
	}

	utils.WriteJSON(writer, http.StatusOK, commondto.SingleDataResponse[*dto.MenuResponse]{
		Title: "Menu item updated successfully",
		Data:  updatedMenu,
	})
}

// DeleteMenuHandler handles the deletion of a menu item
func (mc *MenuController) DeleteMenuHandler(writer http.ResponseWriter, request *http.Request) {
	vars := mux.Vars(request)
	id := vars["id"]

	user := guards.ExtractAuthenticatedUser(request)
	if user == nil {
		errors.ErrorResponse(writer, request, errors.UnauthorizedError("Authentication required"))
		return
	}

	appErr := mc.service.DeleteMenu(request.Context(), user, id)
	if appErr != nil {
		errors.ErrorResponse(writer, request, appErr)
		return
	}

	utils.WriteJSON(writer, http.StatusOK, commondto.MessageResponse{
		Title:   "Menu item deleted successfully",
		Message: "The menu item has been removed from the restaurant.",
	})
}
