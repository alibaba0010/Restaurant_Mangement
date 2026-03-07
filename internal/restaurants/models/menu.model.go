package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/uptrace/bun"
)

type MenuCategory struct {
	bun.BaseModel `bun:"table:menu_categories"`

	ID           uuid.UUID `bun:"type:uuid,pk,default:gen_random_uuid()" json:"id"`
	RestaurantID uuid.UUID `bun:"type:uuid,notnull" json:"restaurant_id"`
	Name         string    `bun:",notnull" json:"name"`
	Description  string    `bun:",nullzero" json:"description,omitempty"`
	SortOrder    int       `bun:",default:0" json:"sort_order"`
	CreatedAt    time.Time `bun:",nullzero,notnull,default:current_timestamp" json:"created_at"`
	UpdatedAt    time.Time `bun:",nullzero,notnull,default:current_timestamp" json:"updated_at"`

	Menus []*Menu `bun:"m2m:menu_category_joins,join:Category=Menu" json:"menus,omitempty"`
}

type MenuCategoryJoin struct {
	bun.BaseModel `bun:"table:menu_category_joins"`

	MenuID     uuid.UUID `bun:"type:uuid,pk"`
	Menu       *Menu     `bun:"rel:belongs-to,join:menu_id=id"`
	CategoryID uuid.UUID `bun:"type:uuid,pk"`
	Category   *MenuCategory `bun:"rel:belongs-to,join:category_id=id"`
}

type Menu struct {
	bun.BaseModel `bun:"table:menu"`

	ID              uuid.UUID       `bun:"type:uuid,pk,default:gen_random_uuid()" json:"id"`
	Name            string          `bun:",notnull" json:"name"`
	Description     string          `bun:",nullzero" json:"description,omitempty"`
	Price           decimal.Decimal `bun:"type:decimal(10,2),notnull" json:"price"`
	ImageURLs       []string        `bun:"image_urls,type:jsonb,nullzero" json:"image_urls,omitempty"`
	VideoURL        string          `bun:"video_url,nullzero" json:"video_url,omitempty"`
	RestaurantID    uuid.UUID       `bun:"type:uuid,notnull" json:"restaurant_id"`
	CategoryIDs     []uuid.UUID     `bun:"-" json:"category_ids,omitempty"` // For internal mapping
	Tags            []string        `bun:"type:jsonb,nullzero" json:"tags,omitempty"`
	IsAvailable     bool            `bun:",notnull,default:true" json:"is_available"`
	PrepTimeMinutes int             `bun:",nullzero" json:"prep_time_minutes,omitempty"`
	Calories        int             `bun:",nullzero" json:"calories,omitempty"`

	// New fields from review
	StockQuantity   int             `bun:",notnull,default:0" json:"stock_quantity"`
	IsVegetarian    bool            `bun:",notnull,default:false" json:"is_vegetarian"`
	IsVegan         bool            `bun:",notnull,default:false" json:"is_vegan"`
	IsGlutenFree    bool            `bun:",notnull,default:false" json:"is_gluten_free"`
	Allergens       []string        `bun:"type:jsonb,nullzero" json:"allergens,omitempty"`

	CreatedAt       time.Time       `bun:",nullzero,notnull,default:current_timestamp" json:"created_at"`
	UpdatedAt       time.Time       `bun:",nullzero,notnull,default:current_timestamp" json:"updated_at"`
	DeletedAt       time.Time       `bun:",soft_delete,nullzero" json:"-"`

	// // Relations
	Restaurant *Restaurant     `bun:"rel:belongs-to,join:restaurant_id=id" json:"restaurant,omitempty"`
	Categories []*MenuCategory `bun:"m2m:menu_category_joins,join:Menu=Category" json:"categories,omitempty"`
}
