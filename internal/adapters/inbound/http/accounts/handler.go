package accounts

import (
	"net/http"
	"strconv"

	httpadapter "github.com/clodoaldomarques/balances-api/internal/adapters/inbound/http"
	accounts "github.com/clodoaldomarques/balances-api/internal/core/accounts/domain"
	"github.com/clodoaldomarques/balances-api/internal/core/accounts/ports"

	"github.com/labstack/echo/v4"
)

// Handler is the HTTP inbound adapter. It only translates transport data and
// delegates every business decision to the injected driving port.
type Handler struct {
	uc ports.AccountsUseCase
}

func New(uc ports.AccountsUseCase) *Handler {
	return &Handler{uc: uc}
}

func (h Handler) CreateNewAccount(c echo.Context) error {
	ctx := c.Request().Context()

	a := new(PostAccountRequest)
	if err := c.Bind(a); err != nil {
		return httpadapter.WriteBadRequest(c, "ERR0001", err)
	}

	if err := c.Validate(a); err != nil {
		return httpadapter.WriteBadRequest(c, "ERR0002", err)
	}

	newAcc, err := h.uc.CreateNewAccount(ctx, a.ToEntity())
	if err != nil {
		return httpadapter.WriteBadRequest(c, "ERR0003", err)
	}

	return c.JSON(http.StatusCreated, AccountToPostAccountResponse(newAcc))
}

func (h Handler) UpdateAccountLimits(c echo.Context) error {
	ctx := c.Request().Context()

	orgID := c.Param("orgID")

	accID, err := strconv.ParseInt(c.Param("accountID"), 10, 64)
	if err != nil {
		return httpadapter.WriteBadRequest(c, "ERR0001", err)
	}

	a := new(PutAccountLimitsRequest)
	if err := c.Bind(a); err != nil {
		return httpadapter.WriteBadRequest(c, "ERR0001", err)
	}

	if err := c.Validate(a); err != nil {
		return httpadapter.WriteBadRequest(c, "ERR0002", err)
	}

	acc, err := h.uc.UpdateAccountLimits(ctx, accID, orgID, a.Limits)
	if err != nil {
		return httpadapter.WriteBadRequest(c, "ERR0003", err)
	}

	return c.JSON(http.StatusOK, AccountToPutAccountResponse(acc))
}

func (h Handler) UpdateAccountStatus(c echo.Context) error {
	ctx := c.Request().Context()

	orgID := c.Param("orgID")

	accID, err := strconv.ParseInt(c.Param("accountID"), 10, 64)
	if err != nil {
		return httpadapter.WriteBadRequest(c, "ERR0001", err)
	}

	a := new(PutAccountStatusRequest)
	if err := c.Bind(a); err != nil {
		return httpadapter.WriteBadRequest(c, "ERR0001", err)
	}

	if err := c.Validate(a); err != nil {
		return httpadapter.WriteBadRequest(c, "ERR0002", err)
	}

	acc, err := h.uc.UpdateAccountStatus(ctx, accID, orgID, accounts.Status(a.Status))
	if err != nil {
		return httpadapter.WriteBadRequest(c, "ERR0003", err)
	}

	return c.JSON(http.StatusOK, AccountToPutAccountResponse(acc))
}

func (h Handler) ProcessEntry(c echo.Context) error {
	ctx := c.Request().Context()

	e := new(PostEntryRequest)
	if err := c.Bind(e); err != nil {
		return httpadapter.WriteBadRequest(c, "ERR0001", err)
	}

	if err := c.Validate(e); err != nil {
		return httpadapter.WriteBadRequest(c, "ERR0002", err)
	}

	acc, err := h.uc.ProcessEntry(ctx, e.ToEntity())
	if err != nil {
		return httpadapter.WriteBadRequest(c, "ERR0003", err)
	}

	return c.JSON(http.StatusOK, AccountToPostAccountResponse(acc))
}
