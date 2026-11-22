package accounts

import (
	"errors"
	"fmt"
	"slices"
	"time"

	accounts "github.com/clodoaldomarques/balances-api/internal/core/accounts/domain"

	"github.com/shopspring/decimal"
)

var (
	validLimits     = []string{accounts.MaxLimit, accounts.TotalLimit, accounts.OverdraftLimit}
	validBalances   = []string{accounts.Available, accounts.Savings, accounts.Blocked}
	validOperations = []string{accounts.Credit, accounts.Debit}
	validStatus     = []string{string(accounts.Active), string(accounts.Inative), string(accounts.OnlyCredit), string(accounts.OnlyDebit)}
	validRules      = []string{accounts.ConsiderAvailableBalance, accounts.ConsiderBlockedBalance, accounts.ConsiderAvailableSavings}
)

type EntityRequest interface {
	Validate() error
}

type PostAccountRequest struct {
	AccountID int64                      `json:"account_id,omitempty"`
	OrgID     string                     `json:"org_id,omitempty"`
	Limits    map[string]decimal.Decimal `json:"limits,omitempty"`
	Balances  map[string]decimal.Decimal `json:"balances,omitempty"`
}

func (p PostAccountRequest) ToEntity() accounts.Account {
	return accounts.Account{
		AccountID: p.AccountID,
		OrgID:     p.OrgID,
		Limits:    p.Limits,
		Balances:  p.Balances,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Status:    accounts.Active,
		Version:   1,
	}
}

func (p PostAccountRequest) Validate() error {
	if len(p.Limits) == 0 {
		return fmt.Errorf("limits can not be empty")
	}
	for k, v := range p.Limits {
		if !slices.Contains(validLimits, k) {
			return errors.New("invalid limit")
		}
		if v.LessThan(decimal.Zero) {
			return fmt.Errorf("limit %s can not less then zero", k)
		}
	}
	if len(p.Balances) == 0 {
		return fmt.Errorf("balances can not be empty")
	}
	for k, v := range p.Balances {
		if !slices.Contains(validBalances, k) {
			return errors.New("invalid balance")
		}
		if v.LessThan(decimal.Zero) {
			return fmt.Errorf("balance %s can not less then zero", k)
		}
	}
	return nil
}

type PostAccountResponse struct {
	AccountID int64                      `json:"account_id,omitempty"`
	OrgID     string                     `json:"org_id,omitempty"`
	Limits    map[string]decimal.Decimal `json:"limits,omitempty"`
	Balances  map[string]decimal.Decimal `json:"balances,omitempty"`
	CreatedAt time.Time                  `json:"created_at,omitempty"`
	UpdatedAt time.Time                  `json:"updated_at,omitempty"`
	Status    string                     `json:"status,omitempty"`
	Version   int64                      `json:"version,omitempty"`
}

func AccountToPostAccountResponse(acc accounts.Account) PostAccountResponse {
	return PostAccountResponse{
		AccountID: acc.AccountID,
		OrgID:     acc.OrgID,
		Limits:    acc.Limits,
		Balances:  acc.Balances,
		CreatedAt: acc.CreatedAt,
		UpdatedAt: acc.UpdatedAt,
		Status:    string(acc.Status),
		Version:   acc.Version,
	}
}

type PutAccountLimitsRequest struct {
	Limits map[string]decimal.Decimal `json:"limits" validate:"required"`
}

func (p PutAccountLimitsRequest) Validate() error {
	if len(p.Limits) == 0 {
		return fmt.Errorf("limits can not be empty")
	}
	for k, v := range p.Limits {
		if !slices.Contains(validLimits, k) {
			return fmt.Errorf("invalid limit: %s", k)
		}
		if v.LessThan(decimal.Zero) {
			return fmt.Errorf("limit %s can not less then zero", k)
		}
	}
	return nil
}

type PutAccountStatusRequest struct {
	Status string `json:"status" validate:"required"`
}

func (p PutAccountStatusRequest) Validate() error {
	if !slices.Contains(validStatus, p.Status) {
		return fmt.Errorf("invalid status: %s", p.Status)
	}
	return nil
}

type PutAccountResponse struct {
	AccountID int64                      `json:"account_id,omitempty"`
	OrgID     string                     `json:"org_id,omitempty"`
	Limits    map[string]decimal.Decimal `json:"limits,omitempty"`
	Balances  map[string]decimal.Decimal `json:"balances,omitempty"`
	CreatedAt time.Time                  `json:"created_at,omitempty"`
	UpdatedAt time.Time                  `json:"updated_at,omitempty"`
	Status    string                     `json:"status,omitempty"`
	Version   int64                      `json:"version,omitempty"`
}

func AccountToPutAccountResponse(acc accounts.Account) PutAccountResponse {
	return PutAccountResponse{
		AccountID: acc.AccountID,
		OrgID:     acc.OrgID,
		Limits:    acc.Limits,
		Balances:  acc.Balances,
		CreatedAt: acc.CreatedAt,
		UpdatedAt: acc.UpdatedAt,
		Status:    string(acc.Status),
		Version:   acc.Version,
	}
}

type PostEntryRequest struct {
	TrackingID string              `json:"tracking_id" validate:"required"`
	AccountID  int64               `json:"account_id" validate:"required"`
	OrgID      string              `json:"org_id" validate:"required"`
	Impacts    []PostImpactRequest `json:"impacts" validate:"required"`
}

func (p PostEntryRequest) ToEntity() accounts.Entry {
	return accounts.Entry{
		TrackingID: p.TrackingID,
		AccountID:  p.AccountID,
		OrgID:      p.OrgID,
		Impacts:    parseToEntity(p.Impacts),
		CreatedAt:  time.Now(),
	}
}

func (p PostEntryRequest) Validate() error {
	for _, i := range p.Impacts {
		if err := i.Validate(); err != nil {
			return err
		}
	}
	return nil
}

type PostImpactRequest struct {
	Balance   string          `json:"balance" validate:"required"`
	Operation string          `json:"operation" validate:"required"`
	Amount    decimal.Decimal `json:"amount" validate:"required"`
	Rules     []string        `json:"rules,omitempty"`
}

func (p PostImpactRequest) Validate() error {
	if !slices.Contains(validBalances, p.Balance) {
		return fmt.Errorf("invalid balance: %s", p.Balance)
	}
	if !slices.Contains(validOperations, p.Operation) {
		return fmt.Errorf("invalid operation: %s", p.Operation)
	}

	for _, r := range p.Rules {
		if !slices.Contains(validRules, r) {
			return fmt.Errorf("invalid consider: %s", r)
		}
	}
	return nil
}

func parseToEntity(impactsRequest []PostImpactRequest) []accounts.Impact {
	impacts := make([]accounts.Impact, 0)
	for _, r := range impactsRequest {
		impact := accounts.Impact{
			Balance:   r.Balance,
			Operation: r.Operation,
			Amount:    r.Amount,
			Rules:     r.Rules,
		}
		impacts = append(impacts, impact)
	}
	return impacts
}

type ErrorResponse struct {
	Code  string `json:"code"`
	Error string `json:"error"`
}
