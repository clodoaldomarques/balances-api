package ports

import (
	"context"

	"github.com/clodoaldomarques/balances-api/internal/core/accounts/domain"
)

//go:generate mockgen -destination=mock.go -package=ports -self_package=github.com/clodoaldomarques/balances-api/internal/core/accounts/ports . AccountsUseCase,AccountRepository,AccountEventPublisher,HealthChecker

// AccountRepository is the driven port for account persistence.
type AccountRepository interface {
	SaveNewAccount(ctx context.Context, a domain.Account) error
	UpdateExistingAccount(ctx context.Context, a domain.Account) error
	RetrieveAccountByID(ctx context.Context, accountID int64, orgID string) (domain.Account, error)
	SaveEntryAndUpdateAccount(ctx context.Context, e domain.Entry, a domain.Account) error
}

// AccountEventPublisher is the driven port for account domain events.
type AccountEventPublisher interface {
	Emit(ctx context.Context, e domain.Event) error
}

// HealthChecker is the driven port used by the HTTP health adapter.
type HealthChecker interface {
	Check(ctx context.Context) error
}
