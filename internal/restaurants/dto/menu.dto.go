package dto

import commondto "github.com/alibaba0010/postgres-api/internal/common/dto"

// CreateMenuInput represents the input for creating a menu item
type CreateMenuInput struct {
	Name            string   `json:"name" validate:"required"`
	Description     string   `json:"description"`
	Price           float64  `json:"price" validate:"required,gt=0"`
	ImageURLs       []string `json:"image_urls"`
	VideoURL        string   `json:"video_url"`
	RestaurantID    string   `json:"restaurant_id" validate:"required,uuid"`
	IsAvailable     bool     `json:"is_available"`
	PrepTimeMinutes int      `json:"prep_time_minutes"`
	Calories        int      `json:"calories"`
}

// UpdateMenuInput represents the input for updating a menu item
type UpdateMenuInput struct {
	Name            *string   `json:"name"`
	Description     *string   `json:"description"`
	Price           *float64  `json:"price" validate:"omitempty,gt=0"`
	ImageURLs       []string  `json:"image_urls"`
	VideoURL        *string   `json:"video_url"`
	IsAvailable     *bool     `json:"is_available"`
	PrepTimeMinutes *int      `json:"prep_time_minutes"`
	Calories        *int      `json:"calories"`
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
	IsAvailable     bool     `json:"is_available"`
	PrepTimeMinutes int      `json:"prep_time_minutes,omitempty"`
	Calories        int      `json:"calories,omitempty"`
	CreatedAt       string   `json:"created_at"`
	UpdatedAt       string   `json:"updated_at"`
}

// MenusListResponse represents the list of menus
type MenusListResponse struct {
	Data []MenuResponse `json:"data"`
	Meta commondto.PaginationMeta `json:"meta"`
}
