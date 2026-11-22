package ports

import (
	"context"

	"github.com/clodoaldomarques/balances-api/internal/core/accounts/domain"
)

//go:generate mockgen -source=outbound.go -destination=mock.go -package=ports
type Repository interface {
	SaveNewAccount(ctx context.Context, a domain.Account) error
	UpdateExistingAccount(ctx context.Context, a domain.Account) error
	RetrieveAccountByID(ctx context.Context, accountID int64, orgID string) (domain.Account, error)
	SaveEntryAndUpdateAccount(ctx context.Context, e domain.Entry, a domain.Account) error
}

type Topic interface {
	Emit(ctx context.Context, e domain.Event) error
}
