package models

import (
	"time"

	"github.com/alibaba0010/postgres-api/internal/common/types"
	"github.com/uptrace/bun"
)

type User struct {
	bun.BaseModel `bun:"table:users"`
	// ID is stored as a UUID in the database. Use string here so Bun
	// doesn't try to scan it into an integer.
	ID        string           `bun:",pk" json:"id"`
	Name      string           `bun:",notnull" json:"name"`
	Email     string           `bun:",unique,notnull" json:"email"`
	Password  string           `bun:",notnull" json:"-"`
	Address   string           `bun:",nullzero" json:"address,omitempty"`
	AvatarURL string           `bun:",nullzero" json:"avatar_url,omitempty"`
	PhoneNumber string         `bun:",nullzero" json:"phone_number,omitempty"`
	Status    types.UserStatus `bun:",notnull,type:user_status,default:'active'" json:"status"`
	Role      types.UserRole   `bun:",notnull,default:'user'" json:"role"`
	CreatedAt time.Time        `bun:",nullzero,notnull,default:current_timestamp" json:"created_at"`
	UpdatedAt time.Time        `bun:",nullzero,notnull,default:current_timestamp" json:"updated_at"`
}


	// // BeforeInsert hook to generate UUIDv7 for ID if not set
	// func (u *User) BeforeInsert(ctx context.Context, _ bun.Query) error {
	//        if u.ID == "" {
	// 	       newUUID, err := utils.GenerateUUIDv7()
	// 	       if err != nil {
	// 		       return err
	// 	       }
	// 	       u.ID = newUUID.String()
	//        }
	//        return nil
	// }
