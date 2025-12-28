package repositories

import (
	"context"
	"fmt"

	"github.com/alibaba0010/postgres-api/internal/auth/models"
	"github.com/alibaba0010/postgres-api/internal/database"
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

// FindAll retrieves users with pagination, filtering, and sorting
func (r *UserRepository) FindAll(ctx context.Context, page, pageSize int, qStr, role, sortBy, order string) ([]models.User, int64, error) {
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
		return nil, 0, err
	}

	sel = sel.Order(fmt.Sprintf("%s %s", sortBy, order)).
		Limit(pageSize).
		Offset((page - 1) * pageSize)

	if err := sel.Scan(ctx); err != nil {
		return nil, 0, err
	}

	return users, int64(total), nil
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

