package mysqldb

import (
	"context"
	"database/sql"

	"github.com/clodoaldomarques/balances-api/config"
	"github.com/clodoaldomarques/core-sdk/pkg/zap/logger"

	_ "github.com/go-sql-driver/mysql"
)

func Connect() (*sql.DB, error) {
	db, err := sql.Open("mysql", config.New().GetMySQLConnectionString())
	if err != nil {
		logger.Fatal(context.Background(), "error on connect database", logger.Fields{
			"error":             err.Error(),
			"connection_string": config.New().GetMySQLConnectionString(),
		})
		return nil, err
	}
	if err = db.Ping(); err != nil {
		db.Close()
		return nil, err
	}
	return db, nil
}
