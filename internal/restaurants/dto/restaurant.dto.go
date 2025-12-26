package dto

// CreateRestaurantInput represents the input for creating a restaurant
type CreateRestaurantInput struct {
	Name        string  `json:"name" validate:"required,min=2,max=100"`
	Description string  `json:"description" validate:"max=500"`
	Address     string  `json:"address" validate:"required,min=5,max=200"`
	CuisineType string  `json:"cuisine_type"`
}

// UpdateRestaurantInput represents the input for updating a restaurant
type UpdateRestaurantInput struct {
	Name        string  `json:"name" validate:"omitempty,min=2,max=100"`
	Description string  `json:"description" validate:"omitempty,max=500"`
	Address     string  `json:"address" validate:"omitempty,min=5,max=200"`
	CuisineType string  `json:"cuisine_type"`
	Rating      *float64 `json:"rating" validate:"omitempty,min=0,max=5"`
}

// RestaurantResponse represents the response structure for a restaurant
type RestaurantResponse struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description,omitempty"`
	Address     string    `json:"address"`
	CuisineType string    `json:"cuisine_type,omitempty"`
	Rating      float64   `json:"rating"`
	CreatedAt   string    `json:"created_at"`
	UpdatedAt   string    `json:"updated_at"`
}

// PaginationMeta provides pagination details
type PaginationMeta struct {
	Page       int   `json:"page"`
	PageSize   int   `json:"page_size"`
	Total      int64 `json:"total"`
	TotalPages int   `json:"total_pages"`
}

// RestaurantsListResponse is the response for listing restaurants
type RestaurantsListResponse struct {
	Title string               `json:"title"`
	Data  []RestaurantResponse `json:"data"`
	Meta  PaginationMeta       `json:"meta"`
}
