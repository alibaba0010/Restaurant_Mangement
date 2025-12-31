package repositories

import (
	"context"
	"fmt"
	"time"

	"github.com/alibaba0010/postgres-api/internal/database"
	"github.com/alibaba0010/postgres-api/internal/restaurants/models"
	"github.com/alibaba0010/postgres-api/internal/utils"
)

type MenuRepository struct{}

var MenuRepo = &MenuRepository{}

func (r *MenuRepository) Create(ctx context.Context, menu *models.Menu) error {
	_, err := database.DB.NewInsert().Model(menu).Exec(ctx)
	return err
}

func (r *MenuRepository) FindByID(ctx context.Context, id string) (*models.Menu, error) {
	menu := new(models.Menu)
	err := database.DB.NewSelect().Model(menu).Where("id = ?", id).Scan(ctx)
	if err != nil {
		return nil, err
	}
	return menu, nil
}

func (r *MenuRepository) FindAll(ctx context.Context, page, pageSize int, queryStr string, restaurantID string, minPrice, maxPrice *float64, isAvailable *bool, sortBy, order string) ([]models.Menu, int64, error) {
	menus := make([]models.Menu, 0)
	sel := database.DB.NewSelect().Model(&menus)

	if queryStr != "" {
		like := "%" + queryStr + "%"
		sel = sel.Where("name ILIKE ? OR description ILIKE ?", like, like)
	}

	if restaurantID != "" {
		sel = sel.Where("restaurant_id = ?", restaurantID)
	}

	if minPrice != nil {
		sel = sel.Where("price >= ?", *minPrice)
	}

	if maxPrice != nil {
		sel = sel.Where("price <= ?", *maxPrice)
	}

	if isAvailable != nil {
		sel = sel.Where("is_available = ?", *isAvailable)
	}

	total, err := sel.Count(ctx)
	if err != nil {
		return nil, 0, err
	}

	// Sanitize and apply sorting
	sortBy, order = utils.SanitizeSort(sortBy, order, []string{"created_at", "price", "name", "calories", "prep_time_minutes"}, "created_at")

	err = sel.Order(fmt.Sprintf("%s %s", sortBy, order)).
		Limit(pageSize).
		Offset((page - 1) * pageSize).
		Scan(ctx)

	if err != nil {
		return nil, 0, err
	}

	return menus, int64(total), nil
}

func (r *MenuRepository) Update(ctx context.Context, menu *models.Menu) error {
	menu.UpdatedAt = time.Now()
	_, err := database.DB.NewUpdate().Model(menu).WherePK().Exec(ctx)
	return err
}
