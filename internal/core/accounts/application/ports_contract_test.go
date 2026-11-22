package application

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/clodoaldomarques/balances-api/internal/core/accounts/domain"
	"github.com/clodoaldomarques/balances-api/internal/core/accounts/ports"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	_ ports.AccountRepository     = (*inMemoryRepository)(nil)
	_ ports.AccountEventPublisher = (*inMemoryPublisher)(nil)
)

func TestServiceWithInMemoryPorts(t *testing.T) {
	t.Run("creates account and emits event", func(t *testing.T) {
		repo := newInMemoryRepository()
		pub := newInMemoryPublisher()
		uc := NewService(repo, pub)

		account, err := uc.CreateNewAccount(context.Background(), buildAccount())

		require.NoError(t, err)
		assert.Equal(t, int64(23052013), account.AccountID)
		require.Len(t, pub.events, 1)
		assert.Equal(t, string(domain.CREATE_ACCOUNT), pub.events[0].EventType())
	})

	t.Run("updates limits and emits event", func(t *testing.T) {
		repo := newInMemoryRepository()
		repo.accounts[accountKey(23052013, "TN-12345678")] = buildAccount()
		pub := newInMemoryPublisher()
		uc := NewService(repo, pub)

		account, err := uc.UpdateAccountLimits(context.Background(), 23052013, "TN-12345678", map[string]decimal.Decimal{
			domain.MaxLimit: decimal.NewFromInt(150),
		})

		require.NoError(t, err)
		assert.Equal(t, decimal.NewFromInt(150), account.Limits[domain.MaxLimit])
		assert.Equal(t, int64(2), account.Version)
		require.Len(t, pub.events, 1)
		assert.Equal(t, string(domain.UPDATE_ACCOUNT), pub.events[0].EventType())
	})

	t.Run("updates status and emits event", func(t *testing.T) {
		repo := newInMemoryRepository()
		repo.accounts[accountKey(23052013, "TN-12345678")] = buildAccount()
		pub := newInMemoryPublisher()
		uc := NewService(repo, pub)

		account, err := uc.UpdateAccountStatus(context.Background(), 23052013, "TN-12345678", domain.Inative)

		require.NoError(t, err)
		assert.Equal(t, domain.Inative, account.Status)
		require.Len(t, pub.events, 1)
		assert.Equal(t, string(domain.UPDATE_ACCOUNT), pub.events[0].EventType())
	})

	t.Run("processes entry and emits event", func(t *testing.T) {
		repo := newInMemoryRepository()
		repo.accounts[accountKey(23052013, "TN-12345678")] = buildAccount()
		pub := newInMemoryPublisher()
		uc := NewService(repo, pub)
		entry := domain.Entry{
			TrackingID: "tracked-entry",
			AccountID:  23052013,
			OrgID:      "TN-12345678",
			Impacts: []domain.Impact{{
				Balance:   domain.Available,
				Operation: domain.Debit,
				Amount:    decimal.NewFromInt(10),
				Rules:     []string{domain.ConsiderAvailableBalance},
			}},
			CreatedAt: time.Now(),
		}

		account, err := uc.ProcessEntry(context.Background(), entry)

		require.NoError(t, err)
		require.Len(t, repo.entries, 1)
		assert.Equal(t, entry.TrackingID, repo.entries[0].TrackingID)
		assert.Equal(t, decimal.NewFromInt(90), account.Balances[domain.Available])
		require.Len(t, pub.events, 1)
		assert.Equal(t, string(domain.PROCESS_ENTRY), pub.events[0].EventType())
	})

	t.Run("propagates repository errors", func(t *testing.T) {
		repo := newInMemoryRepository()
		repo.err = errors.New("repository unavailable")
		pub := newInMemoryPublisher()
		uc := NewService(repo, pub)

		_, err := uc.CreateNewAccount(context.Background(), buildAccount())

		assert.EqualError(t, err, "repository unavailable")
		assert.Empty(t, pub.events)
	})
}

type inMemoryRepository struct {
	accounts map[string]domain.Account
	entries  []domain.Entry
	err      error
}

func newInMemoryRepository() *inMemoryRepository {
	return &inMemoryRepository{accounts: make(map[string]domain.Account)}
}

func (r *inMemoryRepository) SaveNewAccount(ctx context.Context, a domain.Account) error {
	if r.err != nil {
		return r.err
	}
	r.accounts[accountKey(a.AccountID, a.OrgID)] = a
	return nil
}

func (r *inMemoryRepository) UpdateExistingAccount(ctx context.Context, a domain.Account) error {
	if r.err != nil {
		return r.err
	}
	r.accounts[accountKey(a.AccountID, a.OrgID)] = a
	return nil
}

func (r *inMemoryRepository) RetrieveAccountByID(ctx context.Context, accountID int64, orgID string) (domain.Account, error) {
	if r.err != nil {
		return domain.Account{}, r.err
	}
	account, ok := r.accounts[accountKey(accountID, orgID)]
	if !ok {
		return domain.Account{}, errors.New("account not found")
	}
	return account, nil
}

func (r *inMemoryRepository) SaveEntryAndUpdateAccount(ctx context.Context, e domain.Entry, a domain.Account) error {
	if r.err != nil {
		return r.err
	}
	r.entries = append(r.entries, e)
	r.accounts[accountKey(a.AccountID, a.OrgID)] = a
	return nil
}

type inMemoryPublisher struct {
	events []domain.Event
}

func newInMemoryPublisher() *inMemoryPublisher {
	return &inMemoryPublisher{}
}

func (p *inMemoryPublisher) Emit(ctx context.Context, e domain.Event) error {
	p.events = append(p.events, e)
	return nil
}

func accountKey(accountID int64, orgID string) string {
	return fmt.Sprintf("%d/%s", accountID, orgID)
}
