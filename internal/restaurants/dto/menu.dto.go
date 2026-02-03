package dto

import (
	"io"

	commondto "github.com/alibaba0010/postgres-api/internal/common/dto"
)

// CreateMenuInput represents the input for creating a menu item
type CreateMenuInput struct {
	Name            string   `json:"name" validate:"required,min=2,max=100"`
	Description     string   `json:"description" validate:"omitempty,max=500"`
	Price           float64  `json:"price" validate:"required,gt=0"`
	ImageURLs       []string `json:"image_urls" validate:"omitempty,dive,url"`
	VideoURL        string   `json:"video_url" validate:"omitempty,url"`
	RestaurantID    string   `json:"restaurant_id" validate:"required,uuid"`
	CategoryID      string   `json:"category_id" validate:"omitempty,uuid"`
	Tags            []string `json:"tags" validate:"omitempty,dive,min=1"`
	IsAvailable     bool     `json:"is_available"`
	PrepTimeMinutes int      `json:"prep_time_minutes" validate:"omitempty,min=0"`
	Calories        int      `json:"calories" validate:"omitempty,min=0"`
}

// UpdateMenuInput represents the input for updating a menu item
type UpdateMenuInput struct {
	Name            *string  `json:"name" validate:"omitempty,min=2,max=100"`
	Description     *string  `json:"description" validate:"omitempty,max=500"`
	Price           *float64 `json:"price" validate:"omitempty,gt=0"`
	ImageURLs       []string `json:"image_urls" validate:"omitempty,dive,url"`
	VideoURL        *string  `json:"video_url" validate:"omitempty,url"`
	CategoryID      *string  `json:"category_id" validate:"omitempty,uuid"`
	Tags            []string `json:"tags" validate:"omitempty,dive,min=1"`
	IsAvailable     *bool    `json:"is_available"`
	PrepTimeMinutes *int     `json:"prep_time_minutes" validate:"omitempty,min=0"`
	Calories        *int     `json:"calories" validate:"omitempty,min=0"`
}

// MenuResponse represents the response structure for a menu item
type MenuResponse struct {
	ID              string   `json:"id"`
	Name            string   `json:"name"`
	Description     string   `json:"description,omitempty"`
	Price           float64  `json:"price"`
	ImageURLs       []string `json:"image_urls"`
	VideoURL        string   `json:"video_url,omitempty"`
	RestaurantID    string   `json:"restaurant_id"`
	CategoryID      string   `json:"category_id,omitempty"`
	Tags            []string `json:"tags,omitempty"`
	IsAvailable     bool     `json:"is_available"`
	PrepTimeMinutes int      `json:"prep_time_minutes,omitempty"`
	Calories        int      `json:"calories,omitempty"`
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