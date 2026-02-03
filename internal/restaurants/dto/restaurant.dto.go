package dto

import (
	"github.com/alibaba0010/postgres-api/internal/common/address"
	commondto "github.com/alibaba0010/postgres-api/internal/common/dto"
)

// CreateRestaurantInput represents the input for creating a restaurant
type CreateRestaurantInput struct {
	Name              string                `json:"name" validate:"required,min=2,max=100"`
	Description       string                `json:"description" validate:"omitempty,max=500"`
	Address           *address.AddressInput `json:"address"`
	AvatarURL         string                `json:"avatar_url" validate:"omitempty,url"`
	Status            string                `json:"status" validate:"omitempty,oneof=active inactive blocked deleted"`
	UserID            *string               `json:"user_id" validate:"omitempty,uuid"`
	Capacity          *int                  `json:"capacity" validate:"omitempty,min=0,max=10000"`
	DeliveryAvailable *bool                 `json:"delivery_available"`
	TakeawayAvailable *bool                 `json:"takeaway_available"`
}

// UpdateRestaurantInput represents the input for updating a restaurant
type UpdateRestaurantInput struct {
	Name              string                `json:"name" validate:"omitempty,min=2,max=100"`
	Description       string                `json:"description" validate:"omitempty,max=500"`
	Address           *address.AddressInput `json:"address"`
	AvatarURL         string                `json:"avatar_url" validate:"omitempty,url"`
	Status            string                `json:"status" validate:"omitempty,oneof=active inactive blocked deleted"`
	UserID            *string               `json:"user_id" validate:"omitempty,uuid"`
	Capacity          *int                  `json:"capacity" validate:"omitempty,min=0,max=10000"`
	DeliveryAvailable *bool                 `json:"delivery_available"`
	TakeawayAvailable *bool                 `json:"takeaway_available"`
	Rating            *float64              `json:"rating" validate:"omitempty,min=0,max=5"`
}

// RestaurantResponse represents the response structure for a restaurant
type RestaurantResponse struct {
	ID                string  `json:"id"`
	Name              string  `json:"name"`
	Description       string  `json:"description,omitempty"`
	Address           string  `json:"address"`
	AvatarURL         string  `json:"avatar_url,omitempty"`
	Status            string  `json:"status"`
	UserID            *string `json:"user_id,omitempty"`
	Capacity          int     `json:"capacity,omitempty"`
	DeliveryAvailable bool    `json:"delivery_available"`
	TakeawayAvailable bool    `json:"takeaway_available"`
	Rating            float64 `json:"rating"`
	Latitude          float64 `json:"latitude,omitempty"`
	Longitude         float64 `json:"longitude,omitempty"`
	CreatedAt         string  `json:"created_at"`
	UpdatedAt         string  `json:"updated_at"`
}

// RestaurantsListResponse is the response for listing restaurants
type RestaurantsListResponse struct {
	Title string               `json:"title"`
	Data  []RestaurantResponse `json:"data"`
	Meta  commondto.CursorMeta `json:"meta"`
}
