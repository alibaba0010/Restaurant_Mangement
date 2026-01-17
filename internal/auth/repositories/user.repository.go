package repositories

import (
	"context"
	"fmt"

	"github.com/alibaba0010/postgres-api/internal/auth/models"
	"github.com/alibaba0010/postgres-api/internal/database"
	"github.com/alibaba0010/postgres-api/internal/utils"
	"github.com/uptrace/bun"
)

// UserRepository handles database operations for users
type UserRepository struct{}

var UserRepo = &UserRepository{}

func (r *UserRepository) Create(ctx context.Context, user *models.User) error {
	_, err := database.DB.NewInsert().Model(user).Returning("id").Exec(ctx)
	return err
}

func (r *UserRepository) FindByID(ctx context.Context, id string) (*models.User, error) {
	user := new(models.User)
	err := database.DB.NewSelect().Model(user).Where("id = ?", id).Scan(ctx)
	if err != nil {
		return nil, err
	}
	return user, nil
}

func (r *UserRepository) FindByEmail(ctx context.Context, email string) (*models.User, error) {
	user := new(models.User)
	err := database.DB.NewSelect().Model(user).Where("email = ?", email).Scan(ctx)
	if err != nil {
		return nil, err
	}
	return user, nil
}

func (r *UserRepository) ExistsByEmail(ctx context.Context, email string) (bool, error) {
	return database.DB.NewSelect().Model((*models.User)(nil)).Where("email = ?", email).Exists(ctx)
}

func (r *UserRepository) Update(ctx context.Context, user *models.User, columns ...string) error {
	q := database.DB.NewUpdate().Model(user).WherePK()
	if len(columns) > 0 {
		q = q.Column(columns...)
	}
	_, err := q.Exec(ctx)
	return err
}

// FindAll retrieves users with cursor-based pagination
func (r *UserRepository) FindAll(ctx context.Context, limit int, cursorStr, qStr, role, sortBy, order string) ([]models.User, string, bool, int64, error) {
	users := make([]models.User, 0)
	sel := database.DB.NewSelect().Model(&users)

	if qStr != "" {
		like := "%" + qStr + "%"
		sel = sel.Where("name ILIKE ? OR email ILIKE ?", like, like)
	}
	if role != "" {
		sel = sel.Where("role = ?", role)
	}

	total, err := sel.Count(ctx)
	if err != nil {
		return nil, "", false, 0, err
	}

	// Sanitize Sort
	sortBy, order = utils.SanitizeSort(sortBy, order, []string{"created_at", "email", "name", "role"}, "created_at")

	// Apply Cursor Filter
	if cursorStr != "" {
		decoded, err := utils.DecodeCursor(cursorStr)
		if err != nil {
			return nil, "", false, 0, err
		}

		op := ">"
		if order == "DESC" {
			op = "<"
		}

		var cursorVal interface{}
		switch sortBy {
		case "created_at":
			cursorVal = utils.GetCursorValueAsTime(decoded.LastValue)
		default:
			cursorVal = utils.GetCursorValueAsString(decoded.LastValue)
		}

		sel = sel.Where(fmt.Sprintf("(%s, id) %s (?, ?)", sortBy, op), cursorVal, decoded.LastID)
	}

	// Order by Sort Column + ID (tie breaker)
	sel = sel.Order(fmt.Sprintf("%s %s", sortBy, order)).OrderExpr("id " + order)

	// Limit
	err = sel.Limit(limit + 1).Scan(ctx)
	if err != nil {
		return nil, "", false, 0, err
	}

	hasMore := false
	nextCursor := ""
	if len(users) > limit {
		hasMore = true
		users = users[:limit]
		lastItem := users[limit-1]

		var lastVal interface{}
		switch sortBy {
		case "created_at":
			lastVal = lastItem.CreatedAt
		case "email":
			lastVal = lastItem.Email
		case "name":
			lastVal = lastItem.Name
		case "role":
			lastVal = lastItem.Role
		default:
			lastVal = lastItem.CreatedAt
		}
		nextCursor = utils.EncodeCursor(lastVal, lastItem.ID)
	}

	return users, nextCursor, hasMore, int64(total), nil
}

// UpdateInTx updates a user within an existing transaction
func (r *UserRepository) UpdateInTx(ctx context.Context, tx bun.Tx, user *models.User, columns ...string) error {
	q := tx.NewUpdate().Model(user).WherePK()
	if len(columns) > 0 {
		q = q.Column(columns...)
	}
	_, err := q.Exec(ctx)
	return err
}

// UpdatePassword updates a user's password by user ID
func (r *UserRepository) UpdatePassword(ctx context.Context, userID, hashedPassword string) error {
	_, err := database.DB.NewUpdate().
		Model((*models.User)(nil)).
		Set("password = ?", hashedPassword).
		Where("id = ?", userID).
		Exec(ctx)
	return err
}
