package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

type Menu struct {
	bun.BaseModel `bun:"table:menu"`

	ID              uuid.UUID `bun:"type:uuid,pk,default:gen_random_uuid()" json:"id"`
	Name            string    `bun:",notnull" json:"name"`
	Description     string    `bun:",nullzero" json:"description,omitempty"`
	Price           float64   `bun:",notnull" json:"price"`
	ImageURLs       []string  `bun:"image_urls,type:jsonb" json:"image_urls"`
	VideoURL        string    `bun:",nullzero" json:"video_url,omitempty"`
	RestaurantID    uuid.UUID `bun:"type:uuid,notnull" json:"restaurant_id"`
	IsAvailable     bool      `bun:",default:true" json:"is_available"`
	PrepTimeMinutes int       `bun:",nullzero" json:"prep_time_minutes,omitempty"`
	Calories        int       `bun:",nullzero" json:"calories,omitempty"`
	CreatedAt       time.Time `bun:",nullzero,notnull,default:current_timestamp" json:"created_at"`
	UpdatedAt       time.Time `bun:",nullzero,notnull,default:current_timestamp" json:"updated_at"`

	// Relations
	Restaurant *Restaurant `bun:"rel:belongs-to,join:restaurant_id=id" json:"restaurant,omitempty"`
}
