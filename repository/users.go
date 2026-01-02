package repository

import (
	"context"
	"database/sql"
	"fmt"

	"powerbi-access-tool/models"
)

type UserRepository struct {
	db *sql.DB
}

func NewUserRepository(db *sql.DB) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) List(ctx context.Context, filter string, sortField string, sortDir string) ([]models.User, error) {
	query := `SELECT PowerBIUserID, PowerBIUser FROM powerbi.Users`

	var args []interface{}
	if filter != "" {
		query += ` WHERE PowerBIUser LIKE @p1`
		args = append(args, "%"+filter+"%")
	}

	// Validate sort field to prevent SQL injection
	validSortFields := map[string]string{
		"id":    "PowerBIUserID",
		"email": "PowerBIUser",
	}
	dbField, ok := validSortFields[sortField]
	if !ok {
		dbField = "PowerBIUser"
	}

	// Validate sort direction
	if sortDir != "asc" && sortDir != "desc" {
		sortDir = "asc"
	}

	query += fmt.Sprintf(` ORDER BY %s %s`, dbField, sortDir)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query users: %w", err)
	}
	defer rows.Close()

	var users []models.User
	for rows.Next() {
		var u models.User
		if err := rows.Scan(&u.PowerBIUserID, &u.PowerBIUser); err != nil {
			return nil, fmt.Errorf("failed to scan user: %w", err)
		}
		users = append(users, u)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating users: %w", err)
	}

	return users, nil
}

func (r *UserRepository) GetByID(ctx context.Context, id int) (*models.User, error) {
	query := `SELECT PowerBIUserID, PowerBIUser FROM powerbi.Users WHERE PowerBIUserID = @p1`

	var u models.User
	err := r.db.QueryRowContext(ctx, query, id).Scan(&u.PowerBIUserID, &u.PowerBIUser)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	return &u, nil
}

func (r *UserRepository) Create(ctx context.Context, email string) (int, error) {
	query := `INSERT INTO powerbi.Users (PowerBIUser) OUTPUT INSERTED.PowerBIUserID VALUES (@p1)`

	var id int
	err := r.db.QueryRowContext(ctx, query, email).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("failed to create user: %w", err)
	}
	return id, nil
}

func (r *UserRepository) Update(ctx context.Context, id int, email string) error {
	query := `UPDATE powerbi.Users SET PowerBIUser = @p1 WHERE PowerBIUserID = @p2`

	result, err := r.db.ExecContext(ctx, query, email, id)
	if err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("user not found")
	}

	return nil
}

func (r *UserRepository) Delete(ctx context.Context, id int) error {
	// Delete related access records first
	accessQuery := `DELETE FROM powerbi.UserAccess WHERE UserID = @p1`
	_, err := r.db.ExecContext(ctx, accessQuery, id)
	if err != nil {
		return fmt.Errorf("failed to delete user access records: %w", err)
	}

	// Delete the user
	query := `DELETE FROM powerbi.Users WHERE PowerBIUserID = @p1`
	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("user not found")
	}

	return nil
}
