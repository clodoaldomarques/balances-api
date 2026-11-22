package accounts_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/clodoaldomarques/balances-api/internal/adapters/inbound/http/accounts"
	"github.com/clodoaldomarques/balances-api/internal/adapters/inbound/http/server"
	"github.com/clodoaldomarques/balances-api/internal/core/accounts/domain"
	"github.com/labstack/echo/v4"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

const validAccountBody = `{
	"account_id": 1,
	"org_id": "TN-12345678",
	"limits": {"max_limit": 100, "total_limit": 100, "overdraft_limit": 10},
	"balances": {"available": 100, "savings": 0, "blocked": 0}
}`

func TestCreateNewAccountUsesInjectedUseCase(t *testing.T) {
	tests := []struct {
		name       string
		useCaseErr error
		wantStatus int
		wantBody   string
	}{
		{
			name:       "returns created account",
			wantStatus: http.StatusCreated,
			wantBody:   `{"account_id":1,"org_id":"TN-12345678","created_at":"0001-01-01T00:00:00Z","updated_at":"0001-01-01T00:00:00Z","status":"active","version":1}`,
		},
		{
			name:       "maps use case error",
			useCaseErr: errors.New("boom"),
			wantStatus: http.StatusBadRequest,
			wantBody:   `{"code":"ERR0003","error":"boom"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uc := &fakeUseCase{err: tt.useCaseErr}
			s := server.New(accounts.New(uc), &fakeHealthChecker{})

			req := httptest.NewRequest(http.MethodPost, "/accounts", strings.NewReader(validAccountBody))
			req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
			rec := httptest.NewRecorder()

			s.ServeHTTP(rec, req)

			assert.Equal(t, tt.wantStatus, rec.Code)
			assert.Equal(t, 1, uc.createCalls)
			assert.JSONEq(t, tt.wantBody, rec.Body.String())
		})
	}
}

func TestHealthCheckConsumesInjectedChecker(t *testing.T) {
	health := &fakeHealthChecker{}
	s := server.New(accounts.New(&fakeUseCase{}), health)

	rec := httptest.NewRecorder()
	s.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))

	assert.Equal(t, 1, health.calls)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.JSONEq(t, `{"data":"Server is up and running"}`, rec.Body.String())
}

type fakeUseCase struct {
	err         error
	createCalls int
}

func (f *fakeUseCase) CreateNewAccount(ctx context.Context, a domain.Account) (domain.Account, error) {
	f.createCalls++
	if f.err != nil {
		return domain.Account{}, f.err
	}
	return domain.Account{
		AccountID: a.AccountID,
		OrgID:     a.OrgID,
		Status:    domain.Active,
		Version:   1,
	}, nil
}

func (f *fakeUseCase) UpdateAccountLimits(ctx context.Context, accountID int64, orgID string, limits map[string]decimal.Decimal) (domain.Account, error) {
	return domain.Account{}, errors.New("not implemented by fake")
}

func (f *fakeUseCase) UpdateAccountStatus(ctx context.Context, accountID int64, orgID string, status domain.Status) (domain.Account, error) {
	return domain.Account{}, errors.New("not implemented by fake")
}

func (f *fakeUseCase) ProcessEntry(ctx context.Context, e domain.Entry) (domain.Account, error) {
	return domain.Account{}, errors.New("not implemented by fake")
}

type fakeHealthChecker struct {
	calls int
	err   error
}

func (f *fakeHealthChecker) Check(ctx context.Context) error {
	f.calls++
	return f.err
}
