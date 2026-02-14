package repositories

import (
	"context"
	"database/sql"
	"time"

	"github.com/alibaba0010/postgres-api/internal/common/logger"
	"github.com/alibaba0010/postgres-api/internal/restaurants/models"
	"github.com/uptrace/bun"
	"go.uber.org/zap"
)

type CategoryRepository struct {
	db *bun.DB
}

// NewCategoryRepository creates a new category repository instance
func NewCategoryRepository(db *bun.DB) *CategoryRepository {
	return &CategoryRepository{db: db}
}
type CategoryRepositoryInterface interface {
	Create(ctx context.Context, category *models.MenuCategory) error
	FindByID(ctx context.Context, id string) (*models.MenuCategory, error)
	FindByRestaurantID(ctx context.Context, restaurantID string) ([]models.MenuCategory, error)
	Update(ctx context.Context, db bun.IDB, category *models.MenuCategory, columns ...string) error
	Delete(ctx context.Context, id string) error
}

// Create inserts a new menu category into the database
func (r *CategoryRepository) Create(ctx context.Context, category *models.MenuCategory) error {
	_, err := r.db.NewInsert().Model(category).Exec(ctx)
	if err != nil {
		logger.Log.Error("failed to create menu category", zap.Error(err))
		return err
	}
	return nil
}

// FindByID retrieves a menu category by ID
func (r *CategoryRepository) FindByID(ctx context.Context, id string) (*models.MenuCategory, error) {
	category := new(models.MenuCategory)
	err := r.db.NewSelect().Model(category).Where("id = ?", id).Scan(ctx)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, err
		}
		logger.Log.Error("failed to find menu category by id", zap.String("id", id), zap.Error(err))
		return nil, err
	}
	return category, nil
}

// FindByRestaurantID retrieves all menu categories for a restaurant
func (r *CategoryRepository) FindByRestaurantID(ctx context.Context, restaurantID string) ([]models.MenuCategory, error) {
	var categories []models.MenuCategory
	err := r.db.NewSelect().
		Model(&categories).
		Where("restaurant_id = ?", restaurantID).
		Order("sort_order ASC").
		Scan(ctx)
	
	if err != nil {
		logger.Log.Error("failed to find menu categories by restaurant id", zap.String("restaurant_id", restaurantID), zap.Error(err))
		return nil, err
	}
	return categories, err
}

// Update updates an existing menu category
func (r *CategoryRepository) Update(ctx context.Context, db bun.IDB, category *models.MenuCategory, columns ...string) error {
	if db == nil {
		db = r.db
	}
	category.UpdatedAt = time.Now()
	q := db.NewUpdate().Model(category).WherePK()
	if len(columns) > 0 {
		q = q.Column(columns...)
	}
	_, err := q.Exec(ctx)
	if err != nil {
		logger.Log.Error("failed to update menu category", zap.String("id", category.ID.String()), zap.Error(err))
		return err
	}
	return nil
}

// Delete removes a menu category by ID
func (r *CategoryRepository) Delete(ctx context.Context, id string) error {
	_, err := r.db.NewDelete().Model((*models.MenuCategory)(nil)).Where("id = ?", id).Exec(ctx)
	if err != nil {
		logger.Log.Error("failed to delete menu category", zap.String("id", id), zap.Error(err))
		return err
	}
	return nil
}
