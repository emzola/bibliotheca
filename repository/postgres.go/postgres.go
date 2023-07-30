package postgres

import (
	"context"
	"database/sql"
	"time"

	"github.com/emzola/bibliotheca/config"
	_ "github.com/lib/pq"
)

// OpenDBConn creates a PostgreSQl database connection pool.
func OpenDBConn(cfg config.Config) (*sql.DB, error) {
	db, err := sql.Open("postgres", cfg.Database.DSN)
	if err != nil {
		return nil, err
	}
	duration, err := time.ParseDuration(cfg.Database.MaxIdleTime)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(cfg.Database.MaxOpenConns)
	db.SetMaxIdleConns(cfg.Database.MaxIdleConns)
	db.SetConnMaxIdleTime(duration)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	err = db.PingContext(ctx)
	if err != nil {
		return nil, err
	}
	return db, nil
}
