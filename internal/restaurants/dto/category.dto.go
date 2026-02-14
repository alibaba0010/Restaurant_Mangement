package dto

import (
	commondto "github.com/alibaba0010/postgres-api/internal/common/dto"
)

type CreateCategoryInput struct {
	RestaurantID string `json:"restaurant_id" validate:"required,uuid"`
	Name         string `json:"name" validate:"required,min=2,max=100"`
	Description  string `json:"description" validate:"omitempty,max=500"`
	SortOrder    int    `json:"sort_order" validate:"omitempty,min=0"`
}

type UpdateCategoryInput struct {
	Name        *string `json:"name" validate:"omitempty,min=2,max=100"`
	Description *string `json:"description" validate:"omitempty,max=500"`
	SortOrder   *int    `json:"sort_order" validate:"omitempty,min=0"`
}

type CategoryResponse struct {
	ID           string         `json:"id"`
	RestaurantID string         `json:"restaurant_id"`
	Name         string         `json:"name"`
	Description  string         `json:"description,omitempty"`
	SortOrder    int            `json:"sort_order"`
	CreatedAt    string         `json:"created_at"`
	UpdatedAt    string         `json:"updated_at"`
	Menus        []MenuResponse `json:"menus,omitempty"`
}

type CategoryListResponse struct {
	Data []CategoryResponse   `json:"data"`
	Meta commondto.CursorMeta `json:"meta"`
}
