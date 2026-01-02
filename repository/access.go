package repository

import (
	"context"
	"database/sql"
	"fmt"

	"powerbi-access-tool/models"
)

type AccessRepository struct {
	db *sql.DB
}

func NewAccessRepository(db *sql.DB) *AccessRepository {
	return &AccessRepository{db: db}
}

func (r *AccessRepository) ListByUser(ctx context.Context, userID int) ([]models.UserAccess, error) {
	query := `
		SELECT ua.UserAccessID, ua.UserID, ua.Group_Bkey, g.GroupName, ua.CreationDate
		FROM powerbi.UserAccess ua
		INNER JOIN dim.[Group] g ON ua.Group_Bkey = g.Group_Bkey
		WHERE ua.UserID = @p1
		ORDER BY g.GroupName`

	rows, err := r.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to query user access: %w", err)
	}
	defer rows.Close()

	var accessList []models.UserAccess
	for rows.Next() {
		var a models.UserAccess
		if err := rows.Scan(&a.UserAccessID, &a.UserID, &a.GroupBkey, &a.GroupName, &a.CreationDate); err != nil {
			return nil, fmt.Errorf("failed to scan user access: %w", err)
		}
		accessList = append(accessList, a)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating user access: %w", err)
	}

	return accessList, nil
}

func (r *AccessRepository) AddGroups(ctx context.Context, userID int, groupBkeys []int) error {
	if len(groupBkeys) == 0 {
		return nil
	}

	// Insert each group access record
	query := `INSERT INTO powerbi.UserAccess (UserID, Group_Bkey) VALUES (@p1, @p2)`

	for _, groupBkey := range groupBkeys {
		_, err := r.db.ExecContext(ctx, query, userID, groupBkey)
		if err != nil {
			return fmt.Errorf("failed to add group %d for user %d: %w", groupBkey, userID, err)
		}
	}

	return nil
}

func (r *AccessRepository) Remove(ctx context.Context, accessID int) error {
	query := `DELETE FROM powerbi.UserAccess WHERE UserAccessID = @p1`

	result, err := r.db.ExecContext(ctx, query, accessID)
	if err != nil {
		return fmt.Errorf("failed to remove access: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("access record not found")
	}

	return nil
}

func (r *AccessRepository) Exists(ctx context.Context, userID int, groupBkey int) (bool, error) {
	query := `SELECT COUNT(1) FROM powerbi.UserAccess WHERE UserID = @p1 AND Group_Bkey = @p2`

	var count int
	err := r.db.QueryRowContext(ctx, query, userID, groupBkey).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("failed to check access existence: %w", err)
	}

	return count > 0, nil
}
