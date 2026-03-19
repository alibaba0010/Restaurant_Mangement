package models

import (
	"time"

	"github.com/alibaba0010/postgres-api/internal/common/address"
	"github.com/alibaba0010/postgres-api/internal/common/types"
	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

type User struct {
	bun.BaseModel `bun:"table:users"`

	ID          string           `bun:",pk" json:"id"`
	Name        string           `bun:",notnull" json:"name"`
	Email       string           `bun:",unique,notnull" json:"email"`
	Password    string           `bun:",notnull" json:"-"`
	AddressID   *uuid.UUID       `bun:"type:uuid,nullzero" json:"address_id,omitempty"`
	Addresses   []*address.AddressModel `bun:"rel:has-many,join:id=user_id" json:"addresses,omitempty"`
	AvatarURL   string           `bun:",nullzero" json:"avatar_url,omitempty"`
	PhoneNumber string           `bun:",nullzero" json:"phone_number,omitempty"`
	Status      types.UserStatus `bun:",notnull,type:user_status,default:'active'" json:"status"`
	Role        types.UserRole   `bun:",notnull,default:'user'" json:"role"`
	CreatedAt   time.Time        `bun:",nullzero,notnull,default:current_timestamp" json:"created_at"`
	UpdatedAt   time.Time        `bun:",nullzero,notnull,default:current_timestamp" json:"updated_at"`
}
