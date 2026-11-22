package mysql

import (
	"context"
	"database/sql"

	"github.com/clodoaldomarques/core-sdk/pkg/zap/logger"

	_ "github.com/go-sql-driver/mysql"
)

// Connect opens and validates a connection using a DSN supplied by the
// composition root. database/sql pools are safe for concurrent use.
func Connect(dsn string) (*sql.DB, error) {
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		logger.Fatal(context.Background(), "error on connect database", logger.Fields{
			"error":             err.Error(),
			"connection_string": dsn,
		})
		return nil, err
	}
	if err = db.Ping(); err != nil {
		db.Close()
		return nil, err
	}
	return db, nil
}
