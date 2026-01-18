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

func (r *MenuRepository) FindAll(ctx context.Context, limit int, cursorStr string, queryStr string, restaurantID string, minPrice, maxPrice *float64, isAvailable *bool, sortBy, order string) ([]models.Menu, string, bool, int64, error) {
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
		return nil, "", false, 0, err
	}

	// Sanitize Sort
	sortBy, order = utils.SanitizeSort(sortBy, order, []string{"created_at", "price", "name", "calories", "prep_time_minutes"}, "created_at")

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


		var cursorVal utils.CursorValue
		switch sortBy {
		case "created_at":
			cursorVal = utils.GetCursorValueAsTime(decoded.LastValue)
		case "price", "calories", "prep_time_minutes": // numeric
			cursorVal = utils.GetCursorValueAsFloat(decoded.LastValue)
		default:
			cursorVal = utils.GetCursorValueAsString(decoded.LastValue)
		}


		sel = sel.Where(fmt.Sprintf("(%s, id) %s (?, ?)", sortBy, op), cursorVal, decoded.LastID)
	}

	// Order by Sort Column + ID (tie breaker)
	sel = sel.Order(fmt.Sprintf("%s %s", sortBy, order)).OrderExpr("id " + order)

	err = sel.Limit(limit + 1).Scan(ctx)
	if err != nil {
		return nil, "", false, 0, err
	}

	hasMore := false
	nextCursor := ""
	if len(menus) > limit {
		hasMore = true
		menus = menus[:limit]
		lastItem := menus[limit-1]

		var lastVal utils.CursorValue
		switch sortBy {
		case "created_at":
			lastVal = lastItem.CreatedAt
		case "price":
			lastVal = lastItem.Price
		case "name":
			lastVal = lastItem.Name
		case "calories":
			lastVal = lastItem.Calories
		case "prep_time_minutes":
			lastVal = lastItem.PrepTimeMinutes
		default:
			lastVal = lastItem.CreatedAt
		}
		nextCursor = utils.EncodeCursor(lastVal, lastItem.ID.String())
	}

	return menus, nextCursor, hasMore, int64(total), nil
}

func (r *MenuRepository) Update(ctx context.Context, menu *models.Menu) error {
	menu.UpdatedAt = time.Now()
	_, err := database.DB.NewUpdate().Model(menu).WherePK().Exec(ctx)
	return err
}
