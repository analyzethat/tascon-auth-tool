package db

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"

	_ "github.com/microsoft/go-mssqldb"
)

type Config struct {
	Server   string
	Database string
	Username string
	Password string
}

func Open(cfg Config) (*sql.DB, error) {
	query := url.Values{}
	query.Add("database", cfg.Database)

	connStr := fmt.Sprintf("sqlserver://%s:%s@%s?%s",
		url.PathEscape(cfg.Username),
		url.PathEscape(cfg.Password),
		cfg.Server,
		query.Encode())

	db, err := sql.Open("sqlserver", connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	ctx := context.Background()
	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	return db, nil
}
