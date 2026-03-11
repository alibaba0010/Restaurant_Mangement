package repositories

import (
	"context"

	"github.com/alibaba0010/postgres-api/internal/auth/models"
	"github.com/alibaba0010/postgres-api/internal/database"
	"github.com/uptrace/bun"
)

// TokenRepository handles database operations for refresh tokens
type TokenRepository struct{}

var TokenRepo = &TokenRepository{}

func (r *TokenRepository) Create(ctx context.Context, token *models.RefreshToken) error {
	_, err := database.DB.NewInsert().Model(token).Exec(ctx)
	return err
}

func (r *TokenRepository) FindOne(ctx context.Context, userID, token string) (*models.RefreshToken, error) {
	rt := new(models.RefreshToken)
	err := database.DB.NewSelect().Model(rt).
		Where("user_id = ? AND token = ?", userID, token).
		Scan(ctx)
	if err != nil {
		return nil, err
	}
	return rt, nil
}

func (r *TokenRepository) FindByToken(ctx context.Context, token string) (*models.RefreshToken, error) {
	rt := new(models.RefreshToken)
	err := database.DB.NewSelect().Model(rt).
		Where("token = ?", token).
		Scan(ctx)
	if err != nil {
		return nil, err
	}
	return rt, nil
}

func (r *TokenRepository) Exists(ctx context.Context, userID, token string) (bool, error) {
	return database.DB.NewSelect().Model((*models.RefreshToken)(nil)).
		Where("user_id = ? AND token = ?", userID, token).
		Exists(ctx)
}

func (r *TokenRepository) DeleteOne(ctx context.Context, userID, token string) error {
	_, err := database.DB.NewDelete().
		Model((*models.RefreshToken)(nil)).
		Where("user_id = ? AND token = ?", userID, token).
		Exec(ctx)
	return err
}

func (r *TokenRepository) DeleteAllForUser(ctx context.Context, userID string) (int64, error) {
	res, err := database.DB.NewDelete().
		Model((*models.RefreshToken)(nil)).
		Where("user_id = ?", userID).
		Exec(ctx)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

// DeleteAllForUserInTx removes all tokens for a user within a transaction
func (r *TokenRepository) DeleteAllForUserInTx(ctx context.Context, tx bun.Tx, userID string) (int64, error) {
	res, err := tx.NewDelete().
		Model((*models.RefreshToken)(nil)).
		Where("user_id = ?", userID).
		Exec(ctx)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

// RotateToken performs the transactional rotation of a token
func (r *TokenRepository) RotateToken(ctx context.Context, oldUserID, oldToken string, newToken *models.RefreshToken) error {
	return database.DB.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		// Delete the old token
		if oldToken != "" {
			_, err := tx.NewDelete().
				Model((*models.RefreshToken)(nil)).
				Where("user_id = ? AND token = ?", oldUserID, oldToken).
				Exec(ctx)
			if err != nil {
				return err
			}
		}

		// Insert the new token
		_, err := tx.NewInsert().Model(newToken).Exec(ctx)
		return err
	})
}

