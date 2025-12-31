package repositories

import (
	"context"
	"time"
	"fmt"

	"github.com/alibaba0010/postgres-api/internal/database"
	"github.com/alibaba0010/postgres-api/internal/restaurants/models"
	"github.com/alibaba0010/postgres-api/internal/utils"
)

type RestaurantRepository struct{}

var RestaurantRepo = &RestaurantRepository{}

func (r *RestaurantRepository) Create(ctx context.Context, restaurant *models.Restaurant) error {
	_, err := database.DB.NewInsert().Model(restaurant).Exec(ctx)
	return err
}

func (r *RestaurantRepository) FindByID(ctx context.Context, id string) (*models.Restaurant, error) {
	restaurant := new(models.Restaurant)
	err := database.DB.NewSelect().Model(restaurant).Where("id = ?", id).Scan(ctx)
	if err != nil {
		return nil, err
	}
	return restaurant, nil
}

func (r *RestaurantRepository) FindAll(ctx context.Context, page, pageSize int, qStr string, userID *string, sortBy, order string) ([]models.Restaurant, int64, error) {
	restaurants := make([]models.Restaurant, 0)
	sel := database.DB.NewSelect().Model(&restaurants)

	if qStr != "" {
		like := "%" + qStr + "%"
		sel = sel.Where("name ILIKE ? OR description ILIKE ?", like, like)
	}

	if userID != nil {
		sel = sel.Where("user_id = ?", *userID)
	}

	total, err := sel.Count(ctx)
	if err != nil {
		return nil, 0, err
	}

	// Sanitize and apply sorting
	sortBy, order = utils.SanitizeSort(sortBy, order, []string{"created_at", "name", "rating", "capacity"}, "created_at")

	err = sel.Order(fmt.Sprintf("%s %s", sortBy, order)).
		Limit(pageSize).
		Offset((page - 1) * pageSize).
		Scan(ctx)

	if err != nil {
		return nil, 0, err
	}

	return restaurants, int64(total), nil
}

func (r *RestaurantRepository) Update(ctx context.Context, restaurant *models.Restaurant) error {
	restaurant.UpdatedAt = time.Now()
	_, err := database.DB.NewUpdate().Model(restaurant).WherePK().Exec(ctx)
	return err
}
