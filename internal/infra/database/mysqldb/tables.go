package mysqldb

import (
	"encoding/json"
	"time"

	"github.com/clodoaldomarques/balances-api/internal/domain/accounts"
	"github.com/clodoaldomarques/core-sdk/pkg/commons"

	"github.com/shopspring/decimal"
)

type Account struct {
	AccountID int64              `json:"account_id"`
	OrgID     string             `json:"org_id"`
	Limits    commons.DecimalMap `json:"limits"`
	Balances  commons.DecimalMap `json:"balances"`
	CreatedAt time.Time          `json:"created_at"`
	UpdatedAt time.Time          `json:"updated_at"`
	Status    string             `json:"status"`
	Version   int64              `json:"version"`
}

type Entry struct {
	TrackingID string    `json:"tracking_id"`
	AccountID  int64     `json:"account_id"`
	OrgID      string    `json:"org_id"`
	Impacts    []byte    `json:"impacts"`
	CreatedAt  time.Time `json:"created_at"`
}

type Impact struct {
	Balance   string          `json:"balance"`
	Operation string          `json:"operation"`
	Amount    decimal.Decimal `json:"amount"`
	Rules     []string        `json:"rules,omitempty"`
}

func (a Account) toEntity() accounts.Account {
	return accounts.Account{
		AccountID: a.AccountID,
		OrgID:     a.OrgID,
		Limits:    a.Limits,
		Balances:  a.Balances,
		CreatedAt: a.CreatedAt,
		UpdatedAt: a.UpdatedAt,
		Status:    accounts.Status(a.Status),
		Version:   a.Version,
	}
}

func buildAccountTable(a accounts.Account) Account {
	return Account{
		AccountID: a.AccountID,
		OrgID:     a.OrgID,
		Limits:    a.Limits,
		Balances:  a.Balances,
		CreatedAt: a.CreatedAt,
		UpdatedAt: a.UpdatedAt,
		Status:    string(a.Status),
		Version:   a.Version,
	}
}

func buildEntriesTable(e accounts.Entry) (Entry, error) {
	imps, err := json.Marshal(buildImpactsTable(e.Impacts))
	if err != nil {
		return Entry{}, err
	}
	return Entry{
		TrackingID: e.TrackingID,
		AccountID:  e.AccountID,
		OrgID:      e.OrgID,
		Impacts:    imps,
		CreatedAt:  e.CreatedAt,
	}, nil
}

func buildImpactsTable(impact []accounts.Impact) []Impact {
	impacts := make([]Impact, 0, len(impact))
	for _, i := range impact {
		new := Impact{
			Balance:   i.Balance,
			Operation: i.Operation,
			Amount:    i.Amount,
			Rules:     i.Rules,
		}
		impacts = append(impacts, new)
	}
	return impacts
}
