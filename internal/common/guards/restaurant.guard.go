package guards

import (
	"context"

	"github.com/alibaba0010/postgres-api/internal/common/errors"
)

// RestaurantOwnerProvider allows checking for restaurant ownership without direct repo dependency
type RestaurantOwnerProvider interface {
	CheckOwnership(ctx context.Context, restaurantID string, userID string) (bool, error)
}

// AuthorizeRestaurantOwner is a common function to check if a user owns a restaurant
func AuthorizeRestaurantOwner(ctx context.Context, provider RestaurantOwnerProvider, restaurantID string, userID string) *errors.AppError {
	isOwner, err := provider.CheckOwnership(ctx, restaurantID, userID)
	if err != nil {
		return errors.InternalError(err)
	}
	if !isOwner {
		return errors.ForbiddenError("You do not have permission to manage this restaurant's resources")
	}
	return nil
}
