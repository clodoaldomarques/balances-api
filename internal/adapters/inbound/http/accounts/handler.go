package accounts

import (
	"net/http"
	"strconv"

	"github.com/clodoaldomarques/balances-api/config"
	"github.com/clodoaldomarques/balances-api/internal/adapters/outbound/mysql"
	"github.com/clodoaldomarques/balances-api/internal/adapters/outbound/sns"
	service "github.com/clodoaldomarques/balances-api/internal/core/accounts/application"
	accounts "github.com/clodoaldomarques/balances-api/internal/core/accounts/domain"

	"github.com/labstack/echo/v4"
)

func CreateNewAccount(c echo.Context) error {
	ctx := c.Request().Context()
	db, _ := mysql.Connect(config.New().GetMySQLConnectionString())
	r := mysql.NewRepository(ctx, db)
	defer r.Close()

	t := sns.NewPublisher(ctx, config.New())
	defer t.Close(ctx)

	s := service.NewService(r, t)

	a := new(PostAccountRequest)
	if err := c.Bind(a); err != nil {
		return c.JSON(http.StatusBadRequest, ErrorResponse{"ERR0001", err.Error()})
	}

	if err := c.Validate(a); err != nil {
		return c.JSON(http.StatusBadRequest, ErrorResponse{"ERR0002", err.Error()})
	}

	acc := a.ToEntity()

	newAcc, err := s.CreateNewAccount(ctx, acc)
	if err != nil {
		return c.JSON(http.StatusBadRequest, ErrorResponse{"ERR0003", err.Error()})
	}

	resp := AccountToPostAccountResponse(newAcc)
	return c.JSON(http.StatusCreated, resp)
}

func UpdateAccountLimits(c echo.Context) error {
	ctx := c.Request().Context()
	db, _ := mysql.Connect(config.New().GetMySQLConnectionString())
	r := mysql.NewRepository(ctx, db)
	defer r.Close()

	t := sns.NewPublisher(ctx, config.New())
	defer t.Close(ctx)

	s := service.NewService(r, t)

	orgID := c.Param("orgID")

	accID, err := strconv.ParseInt(c.Param("accountID"), 10, 64)
	if err != nil {
		return c.JSON(http.StatusBadRequest, ErrorResponse{"ERR0001", err.Error()})
	}

	a := new(PutAccountLimitsRequest)
	if err := c.Bind(a); err != nil {
		return c.JSON(http.StatusBadRequest, ErrorResponse{"ERR0001", err.Error()})
	}

	if err := c.Validate(a); err != nil {
		return c.JSON(http.StatusBadRequest, ErrorResponse{"ERR0002", err.Error()})
	}

	acc, err := s.UpdateAccountLimits(ctx, accID, orgID, a.Limits)
	if err != nil {
		return c.JSON(http.StatusBadRequest, ErrorResponse{"ERR0003", err.Error()})
	}

	resp := AccountToPutAccountResponse(acc)

	return c.JSON(http.StatusOK, resp)
}

func UpdateAccountStatus(c echo.Context) error {
	ctx := c.Request().Context()
	db, _ := mysql.Connect(config.New().GetMySQLConnectionString())
	r := mysql.NewRepository(ctx, db)
	defer r.Close()

	t := sns.NewPublisher(ctx, config.New())
	defer t.Close(ctx)

	s := service.NewService(r, t)

	orgID := c.Param("orgID")

	accID, err := strconv.ParseInt(c.Param("accountID"), 10, 64)
	if err != nil {
		return c.JSON(http.StatusBadRequest, ErrorResponse{"ERR0001", err.Error()})
	}

	a := new(PutAccountStatusRequest)
	if err := c.Bind(a); err != nil {
		return c.JSON(http.StatusBadRequest, ErrorResponse{"ERR0001", err.Error()})
	}

	if err := c.Validate(a); err != nil {
		return c.JSON(http.StatusBadRequest, ErrorResponse{"ERR0002", err.Error()})
	}

	acc, err := s.UpdateAccountStatus(ctx, accID, orgID, accounts.Status(a.Status))
	if err != nil {
		return c.JSON(http.StatusBadRequest, ErrorResponse{"ERR0003", err.Error()})
	}

	resp := AccountToPutAccountResponse(acc)

	return c.JSON(http.StatusOK, resp)
}

func ProcessEntry(c echo.Context) error {
	ctx := c.Request().Context()
	db, _ := mysql.Connect(config.New().GetMySQLConnectionString())
	r := mysql.NewRepository(ctx, db)
	defer r.Close()

	t := sns.NewPublisher(ctx, config.New())
	defer t.Close(ctx)

	s := service.NewService(r, t)

	e := new(PostEntryRequest)
	if err := c.Bind(e); err != nil {
		return c.JSON(http.StatusBadRequest, ErrorResponse{"ERR0001", err.Error()})
	}

	if err := c.Validate(e); err != nil {
		return c.JSON(http.StatusBadRequest, ErrorResponse{"ERR0002", err.Error()})
	}

	acc, err := s.ProcessEntry(ctx, e.ToEntity())
	if err != nil {
		return c.JSON(http.StatusBadRequest, ErrorResponse{"ERR0003", err.Error()})
	}

	resp := AccountToPostAccountResponse(acc)

	return c.JSON(http.StatusOK, resp)
}
