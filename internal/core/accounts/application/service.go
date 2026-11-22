package application

import (
	"context"

	"github.com/clodoaldomarques/balances-api/internal/core/accounts/domain"
	"github.com/clodoaldomarques/balances-api/internal/core/accounts/ports"
	"github.com/clodoaldomarques/core-sdk/pkg/otel/tracer"
	"github.com/clodoaldomarques/core-sdk/pkg/zap/logger"
	"github.com/shopspring/decimal"
)

var _ ports.AccountsUseCase = (*Service)(nil)

type Service struct {
	rep ports.AccountRepository
	pub ports.AccountEventPublisher
}

func NewService(r ports.AccountRepository, p ports.AccountEventPublisher) *Service {
	return &Service{
		rep: r,
		pub: p,
	}
}

func (s Service) CreateNewAccount(ctx context.Context, a domain.Account) (domain.Account, error) {
	span, ctx := tracer.NewSpanFromContext(ctx, "Service::CreateNewAccount", map[string]any{"account_id": a.AccountID})
	defer span.End()

	if err := s.rep.SaveNewAccount(ctx, a); err != nil {
		span.SetError(err)
		logger.Error(ctx, "error on create new account", logger.Fields{"account": a, "error": err.Error()})
		return domain.Account{}, err
	}

	evt := buildCreateAccountEvent(a)
	s.pub.Emit(ctx, evt)

	return a, nil
}

func (s Service) UpdateAccountLimits(ctx context.Context, accountID int64, orgID string, limits map[string]decimal.Decimal) (domain.Account, error) {
	span, ctx := tracer.NewSpanFromContext(ctx, "Service::UpdateAccountLimits", map[string]any{"account_id": accountID})
	defer span.End()

	acc, err := s.rep.RetrieveAccountByID(ctx, accountID, orgID)
	if err != nil {
		span.SetError(err)
		return domain.Account{}, err
	}

	for limit, value := range limits {
		if err = acc.ChangeLimit(limit, value); err != nil {
			span.SetError(err)
			return domain.Account{}, err
		}
	}

	if err = s.rep.UpdateExistingAccount(ctx, acc); err != nil {
		span.SetError(err)
		return domain.Account{}, err
	}

	evt := buildUpdateAccountEvent(acc)
	s.pub.Emit(ctx, evt)

	return acc, nil

}

func (s Service) UpdateAccountStatus(ctx context.Context, accountID int64, orgID string, status domain.Status) (domain.Account, error) {
	span, ctx := tracer.NewSpanFromContext(ctx, "Service::UpdateAccountStatus", map[string]any{"account_id": accountID})
	defer span.End()

	acc, err := s.rep.RetrieveAccountByID(ctx, accountID, orgID)
	if err != nil {
		span.SetError(err)
		return domain.Account{}, err
	}

	acc.ChangeStatus(status)

	if err = s.rep.UpdateExistingAccount(ctx, acc); err != nil {
		span.SetError(err)
		return domain.Account{}, err
	}

	evt := buildUpdateAccountEvent(acc)
	s.pub.Emit(ctx, evt)

	return acc, nil
}

func (s Service) ProcessEntry(ctx context.Context, e domain.Entry) (domain.Account, error) {
	span, ctx := tracer.NewSpanFromContext(ctx, "Service::ProcessEntry", map[string]any{"account_id": e.AccountID})
	defer span.End()

	acc, err := s.rep.RetrieveAccountByID(ctx, e.AccountID, e.OrgID)
	if err != nil {
		span.SetError(err)
		span.AddAttributes(map[string]any{
			"entry": e,
		})
		return domain.Account{}, err
	}

	if err = acc.ChangeBalances(e.Impacts); err != nil {
		span.SetError(err)
		span.AddAttributes(map[string]any{
			"entry": e,
		})
		return domain.Account{}, err
	}

	if err = s.rep.SaveEntryAndUpdateAccount(ctx, e, acc); err != nil {
		span.SetError(err)
		span.AddAttributes(map[string]any{
			"entry": e,
		})
		return domain.Account{}, err
	}

	evt := buildProcessEntryEvent(acc, e)
	s.pub.Emit(ctx, evt)

	return acc, nil
}

func buildCreateAccountEvent(a domain.Account) domain.Event {
	return domain.CreateAccountEvent{
		AccountID: a.AccountID,
		OrgID:     a.OrgID,
		Limits:    a.Limits,
		Balances:  a.Balances,
		CreatedAt: a.CreatedAt,
		Status:    string(a.Status),
		Version:   a.Version,
	}
}

func buildUpdateAccountEvent(a domain.Account) domain.Event {
	return domain.UpdateAccountEvent{
		AccountID: a.AccountID,
		OrgID:     a.OrgID,
		Limits:    a.Limits,
		Balances:  a.Balances,
		UpdatedAt: a.UpdatedAt,
		Status:    string(a.Status),
		Version:   a.Version,
	}
}

func buildProcessEntryEvent(a domain.Account, e domain.Entry) domain.Event {
	return domain.ProcessEntryEvent{
		AccountID:  a.AccountID,
		OrgID:      a.OrgID,
		TrackingID: e.TrackingID,
		Impacts:    buildImpactEvents(e.Impacts),
		Limits:     a.Limits,
		Balances:   a.Balances,
		Version:    a.Version,
		CreatedAt:  e.CreatedAt,
	}
}

func buildImpactEvents(impacts []domain.Impact) []domain.ImpactEvent {
	evts := make([]domain.ImpactEvent, 0, len(impacts))
	for _, i := range impacts {
		new := domain.ImpactEvent{
			Balance:   i.Balance,
			Operation: i.Operation,
			Amount:    i.Amount,
			Rules:     i.Rules,
		}
		evts = append(evts, new)
	}
	return evts
}
