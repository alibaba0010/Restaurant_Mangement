package repositories

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/alibaba0010/postgres-api/internal/common/logger"
	"github.com/alibaba0010/postgres-api/internal/restaurants/models"
	"github.com/alibaba0010/postgres-api/internal/utils"
	"github.com/shopspring/decimal"
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
type MenuRespository interface {
	Create(ctx context.Context, menu *models.Menu) error
	FindByID(ctx context.Context, id string) (*models.Menu, error)
	FindByIDs(ctx context.Context, ids []string) ([]models.Menu, error)
	FindAll(ctx context.Context, limit int, cursorStr string, queryStr string, restaurantID string, categoryID string, tags []string, minPrice, maxPrice *decimal.Decimal, isAvailable *bool, sortBy, order string) ([]models.Menu, string, bool, int64, error)
	Update(ctx context.Context, db bun.IDB, menu *models.Menu, columns ...string) error
	Delete(ctx context.Context, id string) error
}
// Create inserts a new menu item and its category mappings within a transaction
func (r *MenuRepository) Create(ctx context.Context, menu *models.Menu) error {
	err := r.db.RunInTx(ctx, &sql.TxOptions{}, func(ctx context.Context, tx bun.Tx) error {
		_, err := tx.NewInsert().Model(menu).Exec(ctx)
		if err != nil {
			return err
		}

		if len(menu.CategoryIDs) > 0 {
			joins := make([]models.MenuCategoryJoin, len(menu.CategoryIDs))
			for i, catID := range menu.CategoryIDs {
				joins[i] = models.MenuCategoryJoin{
					MenuID:     menu.ID,
					CategoryID: catID,
				}
			}
			_, err = tx.NewInsert().Model(&joins).Exec(ctx)
			if err != nil {
				return err
			}
		}
		return nil
	})

	if err != nil {
		logger.Log.Error("failed to create menu item", zap.Error(err))
		return err
	}
	return nil
}

// FindByID retrieves a menu item by ID with its categories
func (r *MenuRepository) FindByID(ctx context.Context, id string) (*models.Menu, error) {
	menu := new(models.Menu)
	err := r.db.NewSelect().
		Model(menu).
		Relation("Categories").
		Where("menu.id = ?", id).
		Scan(ctx)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, err
		}
		logger.Log.Error("failed to find menu item by id", zap.String("id", id), zap.Error(err))
		return nil, err
	}
	return menu, nil
}

// FindByIDs retrieves multiple menu items by their IDs with categories
func (r *MenuRepository) FindByIDs(ctx context.Context, ids []string) ([]models.Menu, error) {
	var menus []models.Menu
	if len(ids) == 0 {
		return menus, nil
	}

	err := r.db.NewSelect().
		Model(&menus).
		Relation("Categories").
		Where("menu.id IN (?)", bun.In(ids)).
		Scan(ctx)
	if err != nil {
		logger.Log.Error("failed to find menu items by ids", zap.Strings("ids", ids), zap.Error(err))
		return nil, err
	}
	return menus, nil
}

// FindAll retrieves menu items with various filters and pagination
func (r *MenuRepository) FindAll(ctx context.Context, limit int, cursorStr string, queryStr string, restaurantID string, categoryID string, tags []string, minPrice, maxPrice *decimal.Decimal, isAvailable *bool, sortBy, order string) ([]models.Menu, string, bool, int64, error) {
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

		var cursorVal any
		var castErr error

		switch sortBy {
		case "created_at":
			cursorVal, castErr = utils.GetCursorValueAsTime(decoded.LastValue)
		case "price":
			cursorVal, castErr = utils.GetCursorValueAsFloat(decoded.LastValue)
		case "calories", "prep_time_minutes":
			cursorVal, castErr = utils.GetCursorValueAsInt(decoded.LastValue)
		default:
			cursorVal, castErr = utils.GetCursorValueAsString(decoded.LastValue)
		}

		if castErr != nil {
			return nil, "", false, 0, fmt.Errorf("failed to cast cursor value: %w", castErr)
		}

		sel = sel.Where(fmt.Sprintf("(menu.%s, menu.id) %s (?, ?)", sortBy, op), cursorVal, decoded.LastID)
	}

	// Order by Sort Column + ID (tie breaker)
	sel = sel.Order(fmt.Sprintf("menu.%s %s", sortBy, order)).OrderExpr("menu.id " + order)

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

// Update updates an existing menu item and its category mappings
func (r *MenuRepository) Update(ctx context.Context, db bun.IDB, menu *models.Menu, columns ...string) error {
	if db == nil {
		db = r.db
	}

	err := r.db.RunInTx(ctx, &sql.TxOptions{}, func(ctx context.Context, tx bun.Tx) error {
		menu.UpdatedAt = time.Now()
		updateQuery := tx.NewUpdate().Model(menu).WherePK()
		if len(columns) > 0 {
			updateQuery = updateQuery.Column(columns...)
		}
		_, err := updateQuery.Exec(ctx)
		if err != nil {
			return err
		}

		// If Categories are provided in the model's CategoryIDs field (populated by service)
		// We sync the join table.
		if len(menu.CategoryIDs) > 0 || len(columns) == 0 { // len(columns)==0 means full update
			// Delete existing joins
			_, err = tx.NewDelete().Model((*models.MenuCategoryJoin)(nil)).Where("menu_id = ?", menu.ID).Exec(ctx)
			if err != nil {
				return err
			}

			if len(menu.CategoryIDs) > 0 {
				joins := make([]models.MenuCategoryJoin, len(menu.CategoryIDs))
				for i, catID := range menu.CategoryIDs {
					joins[i] = models.MenuCategoryJoin{
						MenuID:     menu.ID,
						CategoryID: catID,
					}
				}
				_, err = tx.NewInsert().Model(&joins).Exec(ctx)
				if err != nil {
					return err
				}
			}
		}
		return nil
	})

	if err != nil {
		logger.Log.Error("failed to update menu item", zap.String("id", menu.ID.String()), zap.Error(err))
		return err
	}
	return nil
}
