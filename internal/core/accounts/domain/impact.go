package domain

import "github.com/shopspring/decimal"

type Impact struct {
	Balance   string
	Operation string
	Amount    decimal.Decimal
	Rules     []string
}
