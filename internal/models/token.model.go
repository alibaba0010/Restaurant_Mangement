package models

import (
	// "context"
	"time"

	// "github.com/alibaba0010/postgres-api/internal/utils"
	"github.com/uptrace/bun"
)

type RefreshToken struct {
	bun.BaseModel `bun:"table:refresh_tokens"`

	ID        string    `bun:",pk" json:"id"`
	UserID    string    `bun:",notnull" json:"user_id"`
	Token     string    `bun:",unique,notnull" json:"token"`
	IPAddress string    `bun:",nullzero" json:"ip_address"`
	UserAgent string    `bun:",nullzero" json:"user_agent"`
	ExpiresAt time.Time `bun:",notnull" json:"expires_at"`
	CreatedAt time.Time `bun:",nullzero,notnull,default:current_timestamp" json:"created_at"`
}

// BeforeInsert hook to generate UUIDv7 for ID if not set
// func (r *RefreshToken) BeforeInsert(ctx context.Context, _ bun.Query) error {
// 	if r.ID == "" {
// 		newUUID, err := utils.GenerateUUIDv7()
// 		if err != nil {
// 			return err
// 		}
// 		r.ID = newUUID.String()
// 	}
// 	return nil
// }
