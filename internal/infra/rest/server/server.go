package server

import (
	"context"
	"fmt"

	"github.com/clodoaldomarques/balances-api/internal/infra/rest/accounts"
	"github.com/clodoaldomarques/core-sdk/pkg/logger"

	"net/http"

	"github.com/go-playground/validator"
	"github.com/labstack/echo/v4"
)

type Server struct {
	http *echo.Echo
}

func New() *Server {
	s := &Server{
		http: echo.New(),
	}
	s.routes()
	return s
}

func (s *Server) Start(port int) error {
	return s.http.Start(fmt.Sprintf(":%d", port))
}

func (s *Server) routes() {
	s.http.Validator = &CustomValidator{validator: validator.New()}

	// logger interceptor
	s.http.Use(logger.InterceptorWithConfig(logger.InterceptorConfig{
		MaxBodySize:     5 * 1024,
		LogRequestBody:  true,
		LogResponseBody: false, // ligue só para debug
		RedactFields:    []string{"password", "token", "credit_card"},
	}))

	// health check
	s.http.GET("/", HealthCheck)

	// Accounts handler
	s.http.POST("/accounts", accounts.CreateNewAccount)
	s.http.PUT("/accounts/:orgID/:accountID/limits", accounts.UpdateAccountLimits)
	s.http.PUT("/accounts/:orgID/:accountID/status", accounts.UpdateAccountStatus)
	s.http.POST("/accounts/entries", accounts.ProcessEntry)
}

func (s *Server) Shutdown(ctx context.Context) error {
	return s.http.Shutdown(ctx)
}

func HealthCheck(c echo.Context) error {
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
