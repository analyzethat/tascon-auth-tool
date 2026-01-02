package db

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"

	_ "github.com/microsoft/go-mssqldb/azuread"
)

type Config struct {
	Server   string
	Database string
}

func Open(cfg Config) (*sql.DB, error) {
	query := url.Values{}
	query.Add("database", cfg.Database)
	query.Add("fedauth", "ActiveDirectoryAzCli")

	connStr := fmt.Sprintf("sqlserver://%s?%s", cfg.Server, query.Encode())

	db, err := sql.Open("azuresql", connStr)
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
