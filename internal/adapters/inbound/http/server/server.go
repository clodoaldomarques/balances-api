package server

import (
	"context"
	"fmt"
	"net/http"

	"github.com/clodoaldomarques/balances-api/internal/adapters/inbound/http/accounts"
	"github.com/clodoaldomarques/balances-api/internal/core/accounts/ports"
	"github.com/clodoaldomarques/core-sdk/pkg/zap/logger"

	"github.com/go-playground/validator"
	"github.com/labstack/echo/v4"
)

type Server struct {
	http   *echo.Echo
	health ports.HealthChecker
}

func New(accountsHandler *accounts.Handler, health ports.HealthChecker) *Server {
	s := &Server{
		http:   echo.New(),
		health: health,
	}
	s.routes(accountsHandler)
	return s
}

func (s *Server) Start(port int) error {
	return s.http.Start(fmt.Sprintf(":%d", port))
}

func (s Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.http.ServeHTTP(w, r)
}

func (s *Server) routes(accountsHandler *accounts.Handler) {
	s.http.Validator = &CustomValidator{validator: validator.New()}

	// logger interceptor
	s.http.Use(logger.InterceptorWithConfig(logger.InterceptorConfig{
		MaxBodySize:     5 * 1024,
		LogRequestBody:  true,
		LogResponseBody: false, // ligue só para debug
		RedactFields:    []string{"password", "token", "credit_card"},
	}))

	// health check
	s.http.GET("/", s.HealthCheck)

	// Accounts handler
	s.http.POST("/accounts", accountsHandler.CreateNewAccount)
	s.http.PUT("/accounts/:orgID/:accountID/limits", accountsHandler.UpdateAccountLimits)
	s.http.PUT("/accounts/:orgID/:accountID/status", accountsHandler.UpdateAccountStatus)
	s.http.POST("/accounts/entries", accountsHandler.ProcessEntry)
}

func (s *Server) Shutdown(ctx context.Context) error {
	return s.http.Shutdown(ctx)
}

func (s Server) HealthCheck(c echo.Context) error {
	if s.health != nil {
		_ = s.health.Check(c.Request().Context())
	}
	return c.JSON(http.StatusOK, map[string]interface{}{
		"data": "Server is up and running",
	})
}

type CustomValidator struct {
	validator *validator.Validate
}

func (cv *CustomValidator) Validate(i interface{}) error {
	if err := cv.validator.Struct(i); err != nil {
		return err
	}
	r, ok := i.(accounts.EntityRequest)
	if !ok {
		return nil
	}
	if err := r.Validate(); err != nil {
		return err
	}
	return nil
}
