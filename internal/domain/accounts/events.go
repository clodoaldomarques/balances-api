package accounts

import (
	"time"

	"github.com/clodoaldomarques/core-sdk/pkg/commons"
	"github.com/shopspring/decimal"
)

type Type string

var (
	CREATE_ACCOUNT Type = "create_account"
	UPDATE_ACCOUNT Type = "update_account"
	PROCESS_ENTRY  Type = "process_entry"
)

type CreateAccountEvent struct {
	AccountID int64              `json:"account_id"`
	OrgID     string             `json:"org_id"`
	Limits    commons.DecimalMap `json:"limits"`
	Balances  commons.DecimalMap `json:"balances"`
	CreatedAt time.Time          `json:"created_at"`
	Status    string             `json:"status"`
	Version   int64              `json:"version"`
}

func (e CreateAccountEvent) EventType() string {
	return string(CREATE_ACCOUNT)
}

func (e CreateAccountEvent) EventData() any {
	return e
}

type UpdateAccountEvent struct {
	AccountID int64              `json:"account_id"`
	OrgID     string             `json:"org_id"`
	Limits    commons.DecimalMap `json:"limits"`
	Balances  commons.DecimalMap `json:"balances"`
	Status    string             `json:"status"`
	UpdatedAt time.Time          `json:"updated_at"`
	Version   int64              `json:"version"`
}

func (e UpdateAccountEvent) EventType() string {
	return string(UPDATE_ACCOUNT)
}

func (e UpdateAccountEvent) EventData() any {
	return e
}

type ProcessEntryEvent struct {
	AccountID  int64              `json:"account_id"`
	OrgID      string             `json:"org_id"`
	TrackingID string             `json:"tracking_id"`
	Impacts    []ImpactEvent      `json:"impacts"`
	Limits     commons.DecimalMap `json:"limits"`
	Balances   commons.DecimalMap `json:"balances"`
	Version    int64              `json:"version"`
	CreatedAt  time.Time          `json:"created_at"`
}

func (e ProcessEntryEvent) EventType() string {
	return string(PROCESS_ENTRY)
}

func (e ProcessEntryEvent) EventData() any {
	return e
}

type ImpactEvent struct {
	Balance   string          `json:"balance"`
	Operation string          `json:"operation"`
	Amount    decimal.Decimal `json:"amount"`
	Rules     []string        `json:"rules,omitempty"`
}
