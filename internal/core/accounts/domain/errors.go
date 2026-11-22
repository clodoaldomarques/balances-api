package domain

import (
	"fmt"

	"github.com/shopspring/decimal"
)

type ErrInsuficientBalance struct {
	msg string
}

func (e ErrInsuficientBalance) Error() string {
	if e.msg != "" {
		return e.msg
	}
	return "insuficient balance"
}

type ErrAccountDisabled struct {
	msg string
}

func (e ErrAccountDisabled) Error() string {
	if e.msg != "" {
		return e.msg
	}
	return "this Account is disabled for this operation"
}

type ErrValidateLimit struct {
	msg string
}

func (e ErrValidateLimit) Error() string {
	if e.msg != "" {
		return e.msg
	}
	return "this is an invalid operation"
}

type ErrValidateOperation struct {
	operation string
	balance   string
	amount    decimal.Decimal
}

func (e ErrValidateOperation) Error() string {
	return fmt.Sprintf("invalid operation: %s, balance: %s, amount: %s", e.operation, e.balance, e.amount)
}
