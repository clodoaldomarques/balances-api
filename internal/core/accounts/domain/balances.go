package domain

import "github.com/shopspring/decimal"

const (
	Available = "available"
	Savings   = "savings"
	Blocked   = "blocked"
)

const (
	ConsiderAvailableBalance = "consider_available_balance"
	ConsiderAvailableSavings = "consider_available_savings"
	ConsiderBlockedBalance   = "consider_blocked_balance"
)

var rules = map[string]func(*Account, decimal.Decimal) error{
	ConsiderAvailableBalance: validateDebitAvailableBalance,
	ConsiderAvailableSavings: validateDebitSavingsBalance,
	ConsiderBlockedBalance:   validateDebitBlockedBalance,
}

func validateDebitAvailableBalance(a *Account, amount decimal.Decimal) error {
	balance := a.Balances[Available].Add(a.Limits[TotalLimit])
	if balance.LessThan(amount) {
		return ErrInsuficientBalance{}
	}
	return nil
}

func validateDebitSavingsBalance(a *Account, amount decimal.Decimal) error {
	if a.Balances[Savings].LessThan(amount) {
		return ErrInsuficientBalance{}
	}
	return nil
}

func validateDebitBlockedBalance(a *Account, amount decimal.Decimal) error {
	if a.Balances[Blocked].LessThan(amount) {
		return ErrInsuficientBalance{}
	}
	return nil
}
