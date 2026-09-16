package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/clodoaldomarques/balances-api/config"
	"github.com/clodoaldomarques/balances-api/internal/infra/rest/server"
	"github.com/clodoaldomarques/core-sdk/pkg/otel"
	"github.com/clodoaldomarques/core-sdk/pkg/zap/logger"
)

func main() {
	s := server.New()
	go func() {
		c := config.New(config.WithAppPort(5002), config.WithMysqlDBName("balances"))
		if err := s.Start(c.AppPort); err != http.ErrServerClosed {
			logger.Fatal(context.Background(), err.Error(), logger.Fields{})
			os.Exit(1)
		}
	}()

	otel.Start(context.Background())

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	fmt.Println("starting graceful shutdown...")

	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
	defer cancel()

	if err := s.Shutdown(ctx); err != nil {
		fmt.Printf("error on HTTP server shutdown: %v\n", err)
	} else {
		fmt.Println("HTTP server finished with success")
	}

	if err := otel.Shutdown(ctx); err != nil {
		fmt.Printf("error on opentelemetry shutdown: %v\n", err)
	}

	fmt.Println("graceful shutdown done")

}
