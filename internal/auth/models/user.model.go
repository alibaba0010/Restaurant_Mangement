package models

import (
	"time"

	"github.com/alibaba0010/postgres-api/internal/common/types"
	"github.com/uptrace/bun"
)

type User struct {
	bun.BaseModel `bun:"table:users"`

	ID          string           `bun:",pk" json:"id"`
	Name        string           `bun:",notnull" json:"name"`
	Email       string           `bun:",unique,notnull" json:"email"`
	Password    string           `bun:",notnull" json:"-"`
	Address     string           `bun:",nullzero" json:"address,omitempty"`
	AvatarURL   string           `bun:",nullzero" json:"avatar_url,omitempty"`
	PhoneNumber string           `bun:",nullzero" json:"phone_number,omitempty"`
	Status      types.UserStatus `bun:",notnull,type:user_status,default:'active'" json:"status"`
	Role        types.UserRole   `bun:",notnull,default:'user'" json:"role"`
	Latitude    float64          `bun:",nullzero" json:"latitude"`
	Longitude   float64          `bun:",nullzero" json:"longitude"`
	CreatedAt   time.Time        `bun:",nullzero,notnull,default:current_timestamp" json:"created_at"`
	UpdatedAt   time.Time        `bun:",nullzero,notnull,default:current_timestamp" json:"updated_at"`
}
