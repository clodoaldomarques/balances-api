package main

import (
	"context"
	"fmt"
	"net/http"

	"github.com/clodoaldomarques/balances-api/config"
	"github.com/clodoaldomarques/balances-api/internal/infra/rest/server"
	"github.com/clodoaldomarques/core-sdk/pkg/logger"
)

func main() {
	c := config.New(config.WithAppPort(5002), config.WithMysqlDBName("balances"))
	e := server.New()
	if err := e.Start(fmt.Sprintf(":%d", c.AppPort)); err != http.ErrServerClosed {
		logger.Fatal(context.Background(), err.Error(), logger.Fields{})
	}
}
