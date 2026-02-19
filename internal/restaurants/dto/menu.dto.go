package dto

import (
	"io"

	commondto "github.com/alibaba0010/postgres-api/internal/common/dto"
	"github.com/shopspring/decimal"
)

// CreateMenuInput represents the input for creating a menu item
type CreateMenuInput struct {
	Name            string   `json:"name" validate:"required,min=2,max=100"`
	Description     string   `json:"description" validate:"omitempty,max=500"`
	Price           decimal.Decimal `json:"price" validate:"required"`
	ImageURLs       []string `json:"image_urls" validate:"omitempty,dive,url"`
	VideoURL        string   `json:"video_url" validate:"omitempty,url"`
	RestaurantID    string   `json:"restaurant_id" validate:"required,uuid"`
	CategoryIDs     []string `json:"category_ids" validate:"omitempty,max=5,dive,uuid"`
	Tags            []string `json:"tags" validate:"omitempty,dive,min=1"`
	IsAvailable     bool     `json:"is_available"`
	PrepTimeMinutes int      `json:"prep_time_minutes" validate:"omitempty,min=0"`
	Calories        int      `json:"calories" validate:"omitempty,min=0"`
	StockQuantity   int      `json:"stock_quantity" validate:"omitempty,min=0"`
	IsVegetarian    bool     `json:"is_vegetarian"`
	IsVegan         bool     `json:"is_vegan"`
	IsGlutenFree    bool     `json:"is_gluten_free"`
	Allergens       []string `json:"allergens" validate:"omitempty,dive,min=1"`
}

// UpdateMenuInput represents the input for updating a menu item
type UpdateMenuInput struct {
	Name            *string  `json:"name" validate:"omitempty,min=2,max=100"`
	Description     *string  `json:"description" validate:"omitempty,max=500"`
	Price           *decimal.Decimal `json:"price" validate:"omitempty"`
	ImageURLs       []string `json:"image_urls" validate:"omitempty,dive,url"`
	VideoURL        *string  `json:"video_url" validate:"omitempty,url"`
	CategoryIDs     []string `json:"category_ids" validate:"omitempty,max=5,dive,uuid"`
	Tags            []string `json:"tags" validate:"omitempty,dive,min=1"`
	IsAvailable     *bool    `json:"is_available"`
	PrepTimeMinutes *int     `json:"prep_time_minutes" validate:"omitempty,min=0"`
	Calories        *int     `json:"calories" validate:"omitempty,min=0"`
	StockQuantity   *int     `json:"stock_quantity" validate:"omitempty,min=0"`
	IsVegetarian    *bool    `json:"is_vegetarian"`
	IsVegan         *bool    `json:"is_vegan"`
	IsGlutenFree    *bool    `json:"is_gluten_free"`
	Allergens       []string `json:"allergens" validate:"omitempty,dive,min=1"`
}

// MenuResponse represents the response structure for a menu item
type MenuResponse struct {
	ID              string   `json:"id"`
	Name            string   `json:"name"`
	Description     string   `json:"description,omitempty"`
	Price           decimal.Decimal `json:"price"`
	ImageURLs       []string `json:"image_urls"`
	VideoURL        string   `json:"video_url,omitempty"`
	RestaurantID    string             `json:"restaurant_id"`
	Categories      []CategoryResponse `json:"categories,omitempty"`
	Tags            []string           `json:"tags,omitempty"`
	IsAvailable     bool     `json:"is_available"`
	PrepTimeMinutes int      `json:"prep_time_minutes,omitempty"`
	Calories        int      `json:"calories,omitempty"`
	StockQuantity   int      `json:"stock_quantity"`
	IsVegetarian    bool     `json:"is_vegetarian"`
	IsVegan         bool     `json:"is_vegan"`
	IsGlutenFree    bool     `json:"is_gluten_free"`
	Allergens       []string `json:"allergens,omitempty"`
	CreatedAt       string   `json:"created_at"`
	UpdatedAt       string   `json:"updated_at"`
}

// MenusListResponse represents the list of menus
type MenusListResponse struct {
	Data []MenuResponse       `json:"data"`
	Meta commondto.CursorMeta `json:"meta"`
}

// InitiateMultipartUploadInput represents the request to start a multipart upload
type InitiateMultipartUploadInput struct {
	Filename    string `json:"filename" validate:"required"`
	ContentType string `json:"content_type" validate:"required"`
}

// InitiateMultipartUploadResponse represents the response after starting a multipart upload
type InitiateMultipartUploadResponse struct {
	UploadID      string   `json:"upload_id"`
	Key           string   `json:"key"`
	PresignedURLs []string `json:"presigned_urls,omitempty"` // For pre-calculating parts if needed
}
type InitiateMultipartUploadResponseHandler struct {
	Title string 	`json:"title"`
	Data 		 InitiateMultipartUploadResponse `json:"data"`
}
// 
type GenerateMultipartPartURLResponse struct{
	Title string `json:"title"`
	Data string `json:"data"`
}
// CompleteMultipartUploadInput represents the request to finalize a multipart upload
type CompleteMultipartUploadInput struct {
	UploadID string          `json:"upload_id" validate:"required"`
	Key      string          `json:"key" validate:"required"`
	Parts    []CompletedPart `json:"parts" validate:"required"`
}


type CompleteMultipartUploadResponse struct {
	Title string	`json:"title"`
	Data  SingleURLResponse `json:"data"`
}

type SingleURLResponse struct {
    URL string `json:"url"`
}
type GetMenuUploadURLInput struct {
	Filename    string `json:"filename" validate:"required"`
	ContentType string `json:"content_type" validate:"required"`
}
type GetMenuUploadURLResponse struct {
	Title string	`json:"title"`
	Data URLResponse `json:"data"`
}
type URLResponse struct {
	UploadURL string `json:"upload_url"`
	PublicURL string `json:"public_url"`
}
type UploadMenuMediaInput struct {
	Filename    string `json:"filename" validate:"required"`
	ContentType string `json:"content_type" validate:"required"`
	Body        io.Reader `json:"body" validate:"required"`
}

type UploadMenuMediaResponse struct {
	Title string	`json:"title"`
	Data  SingleURLResponse `json:"data"`
}

// CompletedPart represents a single part of a multipart upload
type CompletedPart struct {
	PartNumber int32  `json:"part_number"`
	ETag       string `json:"etag"`
}

type GetMenuByIDResponse struct {
	Title string	`json:"title"`
	Response MenuResponse `json:"response"`
}

// ListMenusFilter represents the filters for listing menu items
type ListMenusFilter struct {
	RestaurantID string           `json:"restaurant_id"`
	CategoryID   string           `json:"category_id"`
	Tags         []string         `json:"tags"`
	MinPrice     *decimal.Decimal `json:"min_price"`
	MaxPrice     *decimal.Decimal `json:"max_price"`
	IsAvailable  *bool            `json:"is_available"`
	Limit        int              `json:"limit"`
	Cursor       string           `json:"cursor"`
	Query        string           `json:"query"`
	SortBy       string           `json:"sort_by"`
	Order        string           `json:"order"`
}