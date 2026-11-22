package domain

import "github.com/shopspring/decimal"

const (
	Credit = "CREDIT"
	Debit  = "DEBIT"
)

var Operations = map[string]func(a *Account, balance string, amount decimal.Decimal) error{
	Debit:  debitFun,
	Credit: creditFun,
}

func creditFun(a *Account, balance string, amount decimal.Decimal) error {
	if a.Status != Active && a.Status != OnlyCredit {
		return ErrValidateOperation{
			operation: Credit,
			balance:   balance,
			amount:    amount,
		}
	}
	a.Balances[balance] = a.Balances[balance].Add(amount)
	return nil
}

func debitFun(a *Account, balance string, amount decimal.Decimal) error {
	if a.Status != Active && a.Status != OnlyDebit {
		return ErrValidateOperation{
			operation: Debit,
			balance:   balance,
			amount:    amount,
		}
	}
	a.Balances[balance] = a.Balances[balance].Sub(amount)
	return nil
}
