package repositories

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/alibaba0010/postgres-api/internal/common/logger"
	"github.com/alibaba0010/postgres-api/internal/restaurants/models"
	"github.com/alibaba0010/postgres-api/internal/utils"
	"github.com/uptrace/bun"
	"go.uber.org/zap"
)

type MenuRepository struct {
	db *bun.DB
}

// NewMenuRepository creates a new menu repository instance
func NewMenuRepository(db *bun.DB) *MenuRepository {
	return &MenuRepository{db: db}
}

// Create inserts a new menu item into the database
func (r *MenuRepository) Create(ctx context.Context, menu *models.Menu) error {
	_, err := r.db.NewInsert().Model(menu).Exec(ctx)
	if err != nil {
		logger.Log.Error("failed to create menu item", zap.Error(err))
		return err
	}
	return nil
}

// FindByID retrieves a menu item by ID
func (r *MenuRepository) FindByID(ctx context.Context, id string) (*models.Menu, error) {
	menu := new(models.Menu)
	err := r.db.NewSelect().Model(menu).Where("id = ?", id).Scan(ctx)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, err
		}
		logger.Log.Error("failed to find menu item by id", zap.String("id", id), zap.Error(err))
		return nil, err
	}
	return menu, nil
}

// FindByIDs retrieves multiple menu items by their IDs
func (r *MenuRepository) FindByIDs(ctx context.Context, ids []string) ([]models.Menu, error) {
	var menus []models.Menu
	if len(ids) == 0 {
		return menus, nil
	}

	err := r.db.NewSelect().Model(&menus).Where("id IN (?)", bun.In(ids)).Scan(ctx)
	if err != nil {
		logger.Log.Error("failed to find menu items by ids", zap.Strings("ids", ids), zap.Error(err))
		return nil, err
	}
	return menus, nil
}

// FindAll retrieves menu items with various filters and pagination
func (r *MenuRepository) FindAll(ctx context.Context, limit int, cursorStr string, queryStr string, restaurantID string, categoryID string, tags []string, minPrice, maxPrice *float64, isAvailable *bool, sortBy, order string) ([]models.Menu, string, bool, int64, error) {
	// Sanitize Limit
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	menus := make([]models.Menu, 0)
	sel := r.db.NewSelect().Model(&menus)

	if queryStr != "" {
		like := "%" + queryStr + "%"
		sel = sel.Where("name ILIKE ? OR description ILIKE ?", like, like)
	}

	if restaurantID != "" {
		sel = sel.Where("restaurant_id = ?", restaurantID)
	}

	if categoryID != "" {
		sel = sel.Where("category_id = ?", categoryID)
	}

	if len(tags) > 0 {
		sel = sel.Where("tags @> ?::jsonb", tags)
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
	
	countSel := sel.Clone()
	total, err := countSel.Count(ctx)
	if err != nil {
		logger.Log.Error("failed to count menu items", zap.Error(err))
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

		// Validate cursor sort
		if decoded.Sort != sortBy {
			return nil, "", false, 0, fmt.Errorf("cursor sort parameter mismatch")
		}

		op := ">"
		if order == "DESC" {
			op = "<"
		}

		var cursorVal utils.CursorValue
		switch sortBy {
		case "created_at":
			cursorVal = utils.GetCursorValueAsTime(decoded.LastValue)
		case "price":
			cursorVal = utils.GetCursorValueAsFloat(decoded.LastValue)
		case "calories", "prep_time_minutes":
			cursorVal = utils.GetCursorValueAsInt(decoded.LastValue)
		default:
			cursorVal = utils.GetCursorValueAsString(decoded.LastValue)
		}

		sel = sel.Where(fmt.Sprintf("(%s, id) %s (?, ?)", sortBy, op), cursorVal, decoded.LastID)
	}

	// Order by Sort Column + ID (tie breaker)
	sel = sel.Order(fmt.Sprintf("%s %s", sortBy, order)).OrderExpr("id " + order)

	err = sel.Limit(limit + 1).Scan(ctx)
	if err != nil {
		logger.Log.Error("failed to list menu items", zap.Error(err))
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
		nextCursor = utils.EncodeCursor(lastVal, lastItem.ID.String(), sortBy)
	}

	return menus, nextCursor, hasMore, int64(total), nil
}

// Update updates an existing menu item
func (r *MenuRepository) Update(ctx context.Context, db bun.IDB, menu *models.Menu, columns ...string) error {
	if db == nil {
		db = r.db
	}
	menu.UpdatedAt = time.Now()
	q := db.NewUpdate().Model(menu).WherePK()
	if len(columns) > 0 {
		q = q.Column(columns...)
	}
	_, err := q.Exec(ctx)
	if err != nil {
		logger.Log.Error("failed to update menu item", zap.String("id", menu.ID.String()), zap.Error(err))
		return err
	}
	return nil
}
