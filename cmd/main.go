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
	"github.com/clodoaldomarques/balances-api/internal/adapters/inbound/http/accounts"
	"github.com/clodoaldomarques/balances-api/internal/adapters/inbound/http/server"
	"github.com/clodoaldomarques/balances-api/internal/adapters/outbound/mysql"
	snsadapter "github.com/clodoaldomarques/balances-api/internal/adapters/outbound/sns"
	"github.com/clodoaldomarques/balances-api/internal/core/accounts/application"
	"github.com/clodoaldomarques/balances-api/internal/core/accounts/ports"
	"github.com/clodoaldomarques/core-sdk/pkg/otel"
	"github.com/clodoaldomarques/core-sdk/pkg/zap/logger"
)

// healthCheckers composes every outbound health adapter into the single port
// consumed by the HTTP adapter.
type healthCheckers []ports.HealthChecker

func (h healthCheckers) Check(ctx context.Context) error {
	for _, checker := range h {
		if err := checker.Check(ctx); err != nil {
			return err
		}
	}
	return nil
}

func main() {
	ctx := context.Background()

	c := config.New(config.WithAppPort(5002), config.WithMysqlDBName("balances"))

	db, err := mysql.Connect(c.GetMySQLConnectionString())
	if err != nil {
		logger.Error(ctx, "error on connect database", logger.Fields{"error": err.Error()})
		os.Exit(1)
	}

	repository := mysql.NewRepository(ctx, db)
	publisher := snsadapter.NewPublisher(ctx, c)
	useCase := application.NewService(repository, publisher)

	accountsHandler := accounts.New(useCase)
	s := server.New(accountsHandler, healthCheckers{repository, publisher})

	go func() {
		if err := s.Start(c.AppPort); err != http.ErrServerClosed {
			logger.Fatal(ctx, err.Error(), logger.Fields{})
			os.Exit(1)
		}
	}()

	otel.Start(ctx)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	fmt.Println("starting graceful shutdown...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
	defer cancel()

	if err := s.Shutdown(shutdownCtx); err != nil {
		fmt.Printf("error on HTTP server shutdown: %v\n", err)
	} else {
		fmt.Println("HTTP server finished with success")
	}

	repository.Close()
	publisher.Close(ctx)

	if err := otel.Shutdown(shutdownCtx); err != nil {
		fmt.Printf("error on opentelemetry shutdown: %v\n", err)
	}

	fmt.Println("graceful shutdown done")
}
