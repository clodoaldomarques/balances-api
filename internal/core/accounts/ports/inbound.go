package ports

import (
	"context"

	"github.com/clodoaldomarques/balances-api/internal/core/accounts/domain"
	"github.com/shopspring/decimal"
)

// AccountsUseCase is the driving port consumed by every inbound adapter.
type AccountsUseCase interface {
	CreateNewAccount(ctx context.Context, a domain.Account) (domain.Account, error)
	UpdateAccountLimits(ctx context.Context, accountID int64, orgID string, limits map[string]decimal.Decimal) (domain.Account, error)
	UpdateAccountStatus(ctx context.Context, accountID int64, orgID string, status domain.Status) (domain.Account, error)
	ProcessEntry(ctx context.Context, e domain.Entry) (domain.Account, error)
}
