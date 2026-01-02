package repository

import (
	"context"
	"database/sql"
	"fmt"

	"powerbi-access-tool/models"
)

type GroupRepository struct {
	db *sql.DB
}

func NewGroupRepository(db *sql.DB) *GroupRepository {
	return &GroupRepository{db: db}
}

func (r *GroupRepository) Search(ctx context.Context, searchTerm string) ([]models.SearchResult, error) {
	if searchTerm == "" {
		return nil, nil
	}

	searchPattern := "%" + searchTerm + "%"

	// First search in level2name
	results, err := r.searchInColumn(ctx, "level2name", searchPattern)
	if err != nil {
		return nil, err
	}

	// If no results in level2name, search in level3name
	if len(results) == 0 {
		results, err = r.searchInColumn(ctx, "level3name", searchPattern)
		if err != nil {
			return nil, err
		}
	}

	return results, nil
}

func (r *GroupRepository) searchInColumn(ctx context.Context, column string, pattern string) ([]models.SearchResult, error) {
	// Using parameterized column name via switch to prevent SQL injection
	var query string
	switch column {
	case "level2name":
		query = `
			SELECT DISTINCT g.Group_Bkey, g.GroupName, 'level2name' as MatchedOn
			FROM dim.[Group] g
			WHERE g.level2name LIKE @p1
			ORDER BY g.GroupName`
	case "level3name":
		query = `
			SELECT DISTINCT g.Group_Bkey, g.GroupName, 'level3name' as MatchedOn
			FROM dim.[Group] g
			WHERE g.level3name LIKE @p1
			ORDER BY g.GroupName`
	default:
		return nil, fmt.Errorf("invalid search column: %s", column)
	}

	rows, err := r.db.QueryContext(ctx, query, pattern)
	if err != nil {
		return nil, fmt.Errorf("failed to search groups in %s: %w", column, err)
	}
	defer rows.Close()

	var results []models.SearchResult
	for rows.Next() {
		var r models.SearchResult
		if err := rows.Scan(&r.GroupBkey, &r.GroupName, &r.MatchedOn); err != nil {
			return nil, fmt.Errorf("failed to scan search result: %w", err)
		}
		results = append(results, r)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating search results: %w", err)
	}

	return results, nil
}

func (r *GroupRepository) GetByBkey(ctx context.Context, groupBkey int) (*models.Group, error) {
	query := `SELECT Group_Bkey, GroupName FROM dim.[Group] WHERE Group_Bkey = @p1`

	var g models.Group
	err := r.db.QueryRowContext(ctx, query, groupBkey).Scan(&g.GroupBkey, &g.GroupName)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get group: %w", err)
	}
	return &g, nil
}
