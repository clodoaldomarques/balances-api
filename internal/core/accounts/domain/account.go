package domain

import (
	"time"

	"github.com/shopspring/decimal"
)

type Status string

const (
	Active     Status = "active"
	OnlyCredit Status = "only_credit"
	OnlyDebit  Status = "only_debit"
	Inative    Status = "inactive"
)

type Account struct {
	AccountID int64
	OrgID     string
	Limits    map[string]decimal.Decimal
	Balances  map[string]decimal.Decimal
	CreatedAt time.Time
	UpdatedAt time.Time
	Status    Status
	Version   int64
}

func (a *Account) ChangeStatus(status Status) {
	a.Status = status
	a.IncreaseVersion()
}

func (a *Account) ChangeLimit(limit string, value decimal.Decimal) error {
	if a.Status == Inative {
		return ErrAccountDisabled{}
	}
	if err := limits[limit](a, value); err != nil {
		return err
	}
	a.Limits[limit] = value
	a.IncreaseVersion()
	return nil
}

func (a *Account) ChangeBalances(impacts []Impact) error {
	for _, i := range impacts {
		if i.Operation == Credit && len(i.Rules) > 0 {
			return ErrValidateOperation{
				operation: i.Operation,
				balance:   i.Balance,
				amount:    i.Amount,
			}
		}
		for _, r := range i.Rules {
			if err := rules[r](a, i.Amount); err != nil {
				return err
			}
		}
		if err := Operations[i.Operation](a, i.Balance, i.Amount); err != nil {
			return err
		}
	}
	a.IncreaseVersion()
	return nil
}

func (a *Account) IncreaseVersion() {
	a.Version++
	a.UpdatedAt = time.Now()
}
