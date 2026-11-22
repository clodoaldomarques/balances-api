package httpadapter

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

// ErrorResponse preserves the HTTP error envelope of the pre-hexagonal API.
type ErrorResponse struct {
	Code  string `json:"code"`
	Error string `json:"error"`
}

// WriteBadRequest maps handled adapter and use-case errors to HTTP 400.
func WriteBadRequest(c echo.Context, code string, err error) error {
	return write(c, http.StatusBadRequest, code, err)
}

// WriteNotFound is the HTTP 404 branch of the mapper. No current domain error
// reaches it; the contract is kept here so inbound adapters share one mapping.
func WriteNotFound(c echo.Context, code string, err error) error {
	return write(c, http.StatusNotFound, code, err)
}

func write(c echo.Context, status int, code string, err error) error {
	return c.JSON(status, ErrorResponse{Code: code, Error: err.Error()})
}
