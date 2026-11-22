package db

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "github.com/lib/pq"
)

type DB struct {
	*sql.DB
}

func Connect(dsn string) (*DB, error) {
	conn, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	conn.SetMaxOpenConns(25)
	conn.SetMaxIdleConns(10)
	conn.SetConnMaxLifetime(15 * time.Minute)

	if err := conn.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &DB{conn}, nil
}

func (db *DB) RunMigrations(migrationDir string) error {
	entries, err := os.ReadDir(migrationDir)
	if err != nil {
		return fmt.Errorf("cannot read migrations directory: %w", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".up.sql") {
			fullPath := filepath.Join(migrationDir, entry.Name())
			content, err := os.ReadFile(fullPath)
			if err != nil {
				return fmt.Errorf("failed reading migration %s: %w", entry.Name(), err)
			}

			log.Printf("Executing migration: %s", entry.Name())
			if _, err := db.Exec(string(content)); err != nil {
				return fmt.Errorf("failed executing migration %s: %w", entry.Name(), err)
			}
		}
	}

	return nil
}
