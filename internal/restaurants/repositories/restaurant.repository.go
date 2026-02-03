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

type RestaurantRepository struct {
	db *bun.DB
}

// NewRestaurantRepository creates a new restaurant repository instance
func NewRestaurantRepository(db *bun.DB) *RestaurantRepository {
	return &RestaurantRepository{db: db}
}

// Create inserts a new restaurant into the database
func (r *RestaurantRepository) Create(ctx context.Context, restaurant *models.Restaurant) error {
	_, err := r.db.NewInsert().Model(restaurant).Exec(ctx)
	if err != nil {
		logger.Log.Error("failed to create restaurant", zap.Error(err))
		return err
	}
	return nil
}

// FindByID retrieves a restaurant by its ID from the database
func (r *RestaurantRepository) FindByID(ctx context.Context, id string) (*models.Restaurant, error) {
	restaurant := new(models.Restaurant)
	err := r.db.NewSelect().Model(restaurant).Where("id = ?", id).Scan(ctx)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, err
		}
		logger.Log.Error("failed to find restaurant by id", zap.String("id", id), zap.Error(err))
		return nil, err
	}
	return restaurant, nil
}

// FindAll retrieves restaurants with cursor-based pagination
func (r *RestaurantRepository) FindAll(ctx context.Context, limit int, cursorStr string, qStr string, userID *string, status *string, sortBy, order string) ([]models.Restaurant, string, bool, int64, error) {
	// 1. Sanitize Limit
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	restaurants := make([]models.Restaurant, 0)
	sel := r.db.NewSelect().Model(&restaurants)

	if qStr != "" {
		like := "%" + qStr + "%"
		sel = sel.Where("name ILIKE ? OR description ILIKE ?", like, like)
	}

	if userID != nil {
		sel = sel.Where("user_id = ?", *userID)
	}

	if status != nil {
		sel = sel.Where("status = ?", *status)
	}

	// Calculate total count
	countSel := sel.Clone()
	total, err := countSel.Count(ctx)
	if err != nil {
		logger.Log.Error("failed to count restaurants", zap.Error(err))
		return nil, "", false, 0, err
	}

	// Sanitize Sort
	sortBy, order = utils.SanitizeSort(sortBy, order, []string{"created_at", "name", "rating", "capacity"}, "created_at")

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

		// Determine the operator based on sort order
		op := ">"
		if order == "DESC" {
			op = "<"
		}

		// Handle type casting for cursor value based on sortBy field
		var cursorVal utils.CursorValue
		switch sortBy {
		case "created_at":
			cursorVal = utils.GetCursorValueAsTime(decoded.LastValue)
		case "rating":
			cursorVal = utils.GetCursorValueAsFloat(decoded.LastValue)
		case "capacity":
			cursorVal = utils.GetCursorValueAsInt(decoded.LastValue)
		default:
			cursorVal = utils.GetCursorValueAsString(decoded.LastValue)
		}

		// Tuple comparison: (column, id) > (value, id)
		sel = sel.Where(fmt.Sprintf("(%s, id) %s (?, ?)", sortBy, op), cursorVal, decoded.LastID)
	}

	// Order by Sort Column + ID (tie breaker)
	sel = sel.Order(fmt.Sprintf("%s %s", sortBy, order)).OrderExpr("id " + order)

	// Fetch limit + 1 to check for next page
	err = sel.Limit(limit + 1).Scan(ctx)
	if err != nil {
		logger.Log.Error("failed to list restaurants", zap.Error(err))
		return nil, "", false, 0, err
	}

	// Check if there are more results
	hasMore := false
	nextCursor := ""
	if len(restaurants) > limit {
		hasMore = true
		restaurants = restaurants[:limit] // Trim the extra item
		lastItem := restaurants[limit-1]

		// Encode next cursor
		var lastVal utils.CursorValue
		switch sortBy {
		case "created_at":
			lastVal = lastItem.CreatedAt
		case "name":
			lastVal = lastItem.Name
		case "rating":
			lastVal = lastItem.Rating
		case "capacity":
			lastVal = lastItem.Capacity
		default:
			lastVal = lastItem.CreatedAt
		}
		nextCursor = utils.EncodeCursor(lastVal, lastItem.ID.String(), sortBy)
	}

	return restaurants, nextCursor, hasMore, int64(total), nil
}

// Update updates an existing restaurant in the database
func (r *RestaurantRepository) Update(ctx context.Context, db bun.IDB, restaurant *models.Restaurant, columns ...string) error {
	if db == nil {
		db = r.db
	}
	restaurant.UpdatedAt = time.Now()
	q := db.NewUpdate().Model(restaurant).WherePK()
	if len(columns) > 0 {
		q = q.Column(columns...)
	}
	_, err := q.Exec(ctx)
	if err != nil {
		logger.Log.Error("failed to update restaurant", zap.String("id", restaurant.ID.String()), zap.Error(err))
		return err
	}
	return nil
}
