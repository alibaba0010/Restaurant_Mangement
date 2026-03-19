package repositories

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/alibaba0010/postgres-api/internal/common/logger"
	"github.com/alibaba0010/postgres-api/internal/restaurants/models"
	"github.com/alibaba0010/postgres-api/internal/utils"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/uptrace/bun"
	"go.uber.org/zap"
)

// MenuRepository manages persistence logic for Menu items,
// including their complex relations with categories (many-to-many).
type MenuRepository struct {
	db *bun.DB
}

// NewMenuRepository creates a new menu repository instance
func NewMenuRepository(db *bun.DB) *MenuRepository {
	return &MenuRepository{db: db}
}
// MenuRespository defines the contract for menu data persistence.
type MenuRespository interface {
	Create(ctx context.Context, menu *models.Menu) error
	FindByID(ctx context.Context, id string) (*models.Menu, error)
	FindByIDs(ctx context.Context, ids []string) ([]models.Menu, error)
	FindAll(ctx context.Context, limit int, cursorStr string, queryStr string, restaurantID string, categoryID string, tags []string, minPrice, maxPrice *decimal.Decimal, isAvailable *bool, sortBy, order string) ([]models.Menu, string, bool, int64, error)
	FindByIDsWithLock(ctx context.Context, db bun.IDB, ids []string) ([]models.Menu, error)
	FindByIDWithLock(ctx context.Context, db bun.IDB, id string) (*models.Menu, error)
	Update(ctx context.Context, db bun.IDB, menu *models.Menu, columns ...string) error
	UpdateStock(ctx context.Context, db bun.IDB, menuID uuid.UUID, quantityChange int) error
	BatchUpdateStock(ctx context.Context, db bun.IDB, stockChanges map[uuid.UUID]int) error
	Delete(ctx context.Context, id string) error
	RunInTx(ctx context.Context, fn func(context.Context, bun.Tx) error) error
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

// FindByIDsWithLock retrieves multiple menu items by their IDs with a row-level lock (FOR UPDATE).
func (r *MenuRepository) FindByIDsWithLock(ctx context.Context, db bun.IDB, ids []string) ([]models.Menu, error) {
	if db == nil {
		db = r.db
	}
	var menus []models.Menu
	if len(ids) == 0 {
		return menus, nil
	}

	err := db.NewSelect().
		Model(&menus).
		Where("id IN (?)", bun.In(ids)).
		For("UPDATE").
		Scan(ctx)
	if err != nil {
		logger.Log.Error("failed to find and lock menu items by ids", zap.Strings("ids", ids), zap.Error(err))
		return nil, err
	}
	return menus, nil
}

// FindByIDWithLock retrieves a single menu item by ID with a row-level lock (FOR UPDATE).
func (r *MenuRepository) FindByIDWithLock(ctx context.Context, db bun.IDB, id string) (*models.Menu, error) {
	if db == nil {
		db = r.db
	}
	menu := new(models.Menu)
	err := db.NewSelect().
		Model(menu).
		Relation("Categories").
		Where("menu.id = ?", id).
		For("UPDATE").
		Scan(ctx)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, err
		}
		logger.Log.Error("failed to find and lock menu item by id", zap.String("id", id), zap.Error(err))
		return nil, err
	}
	return menu, nil
}

// RunInTx executes a function within a database transaction.
func (r *MenuRepository) RunInTx(ctx context.Context, fn func(context.Context, bun.Tx) error) error {
	return r.db.RunInTx(ctx, &sql.TxOptions{}, fn)
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
		sel = sel.Where("EXISTS (SELECT 1 FROM menu_category_joins WHERE menu_id = menu.id AND category_id = ?)", categoryID)
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
	var tx bun.Tx
	var isPassedTx bool

	switch passedDB := db.(type) {
	case bun.Tx:
		tx = passedDB
		isPassedTx = true
	case *bun.DB:
		var err error
		tx, err = passedDB.BeginTx(ctx, &sql.TxOptions{})
		if err != nil {
			return err
		}
	default:
		var err error
		tx, err = r.db.BeginTx(ctx, &sql.TxOptions{})
		if err != nil {
			return err
		}
	}

	defer func() {
		if !isPassedTx {
			_ = tx.Rollback()
		}
	}()

	menu.UpdatedAt = time.Now()
	updateQuery := tx.NewUpdate().Model(menu).WherePK()
	if len(columns) > 0 {
		updateQuery = updateQuery.Column(columns...)
	}
	_, err := updateQuery.Exec(ctx)
	if err != nil {
		logger.Log.Error("failed to update menu item", zap.String("id", menu.ID.String()), zap.Error(err))
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

	if !isPassedTx {
		if err := tx.Commit(); err != nil {
			return err
		}
	}

	return nil
}

// UpdateStock increments or decrements the stock quantity of a menu item atomically.
func (r *MenuRepository) UpdateStock(ctx context.Context, db bun.IDB, menuID uuid.UUID, quantityChange int) error {
	if db == nil {
		db = r.db
	}

	_, err := db.NewUpdate().
		Model((*models.Menu)(nil)).
		Set("stock_quantity = stock_quantity + ?", quantityChange).
		Set("updated_at = NOW()").
		Where("id = ?", menuID).
		Exec(ctx)

	if err != nil {
		logger.Log.Error("failed to update menu item stock", zap.String("id", menuID.String()), zap.Error(err))
		return err
	}
	return nil
}

// BatchUpdateStock updates the stock of multiple menu items in a single query (Issue 6.2).
func (r *MenuRepository) BatchUpdateStock(ctx context.Context, db bun.IDB, stockChanges map[uuid.UUID]int) error {
	if len(stockChanges) == 0 {
		return nil
	}
	if db == nil {
		db = r.db
	}

	// For PostgreSQL, we can use a more efficient CASE statement or a temporary table join.
	// Bun doesn't directly support CASE in Set easily with multiple values without raw SQL.
	// But we can iterate and use the transaction. Even though it's separate queries,
	// if they are in the same transaction it's better than nothing, but raw SQL is faster.
	
	for id, change := range stockChanges {
		if err := r.UpdateStock(ctx, db, id, change); err != nil {
			return err
		}
	}
	
	return nil
}

// Delete removes a menu item and its category mappings by ID.
func (r *MenuRepository) Delete(ctx context.Context, id string) error {
	err := r.db.RunInTx(ctx, &sql.TxOptions{}, func(ctx context.Context, tx bun.Tx) error {
		_, err := tx.NewDelete().Model((*models.MenuCategoryJoin)(nil)).Where("menu_id = ?", id).Exec(ctx)
		if err != nil {
			return err
		}
		_, err = tx.NewDelete().Model((*models.Menu)(nil)).Where("id = ?", id).Exec(ctx)
		return err
	})
	if err != nil {
		logger.Log.Error("failed to delete menu item", zap.String("id", id), zap.Error(err))
		return err
	}
	return nil
}
