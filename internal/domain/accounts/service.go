package accounts

import (
	"context"

	"github.com/clodoaldomarques/core-sdk/pkg/tracer"
	"github.com/shopspring/decimal"
	"go.opentelemetry.io/otel/attribute"
)

type Service struct {
	rep Repository
	pub Topic
}

func NewService(r Repository, p Topic) *Service {
	return &Service{
		rep: r,
		pub: p,
	}
}

func (s Service) CreateNewAccount(ctx context.Context, a Account) (Account, error) {
	span, ctx := tracer.NewSpanFromContext(ctx, "Service::CreateNewAccount", attribute.Int64("account_id", a.AccountID))
	defer span.End()

	if err := s.rep.SaveNewAccount(ctx, a); err != nil {
		span.SetError(err)
		return Account{}, err
	}

	evt := buildCreateAccountEvent(a)
	s.pub.Emit(ctx, evt)

	return a, nil
}

func (s Service) UpdateAccountLimits(ctx context.Context, accountID int64, orgID string, limits map[string]decimal.Decimal) (Account, error) {
	span, ctx := tracer.NewSpanFromContext(ctx, "Service::UpdateAccountLimits", attribute.Int64("account_id", accountID))
	defer span.End()

	acc, err := s.rep.RetrieveAccountByID(ctx, accountID, orgID)
	if err != nil {
		span.SetError(err)
		return Account{}, err
	}

	for limit, value := range limits {
		if err = acc.ChangeLimit(limit, value); err != nil {
			span.SetError(err)
			return Account{}, err
		}
	}

	if err = s.rep.UpdateExistingAccount(ctx, acc); err != nil {
		span.SetError(err)
		return Account{}, err
	}

	evt := buildUpdateAccountEvent(acc)
	s.pub.Emit(ctx, evt)

	return acc, nil

}

func (s Service) UpdateAccountStatus(ctx context.Context, accountID int64, orgID string, status Status) (Account, error) {
	span, ctx := tracer.NewSpanFromContext(ctx, "Service::UpdateAccountStatus", attribute.Int64("account_id", accountID))
	defer span.End()

	acc, err := s.rep.RetrieveAccountByID(ctx, accountID, orgID)
	if err != nil {
		span.SetError(err)
		return Account{}, err
	}

	acc.ChangeStatus(status)

	if err = s.rep.UpdateExistingAccount(ctx, acc); err != nil {
		span.SetError(err)
		return Account{}, err
	}

	evt := buildUpdateAccountEvent(acc)
	s.pub.Emit(ctx, evt)

	return acc, nil
}

func (s Service) ProcessEntry(ctx context.Context, e Entry) (Account, error) {
	span, ctx := tracer.NewSpanFromContext(ctx, "Service::ProcessEntry", attribute.Int64("account_id", e.AccountID))
	defer span.End()

	acc, err := s.rep.RetrieveAccountByID(ctx, e.AccountID, e.OrgID)
	if err != nil {
		span.SetError(err)
		span.AddAttributes(tracer.Attributes{
			"entry": e,
		})
		return Account{}, err
	}

	if err = acc.ChangeBalances(e.Impacts); err != nil {
		span.SetError(err)
		span.AddAttributes(tracer.Attributes{
			"entry": e,
		})
		return Account{}, err
	}

	if err = s.rep.SaveEntryAndUpdateAccount(ctx, e, acc); err != nil {
		span.SetError(err)
		span.AddAttributes(tracer.Attributes{
			"entry": e,
		})
		return Account{}, err
	}

	evt := buildProcessEntryEvent(acc, e)
	s.pub.Emit(ctx, evt)

	return acc, nil
}

func buildCreateAccountEvent(a Account) Event {
	return CreateAccountEvent{
		AccountID: a.AccountID,
		OrgID:     a.OrgID,
		Limits:    a.Limits,
		Balances:  a.Balances,
		CreatedAt: a.CreatedAt,
		Status:    string(a.Status),
		Version:   a.Version,
	}
}

func buildUpdateAccountEvent(a Account) Event {
	return UpdateAccountEvent{
		AccountID: a.AccountID,
		OrgID:     a.OrgID,
		Limits:    a.Limits,
		Balances:  a.Balances,
		UpdatedAt: a.UpdatedAt,
		Status:    string(a.Status),
		Version:   a.Version,
	}
}

func buildProcessEntryEvent(a Account, e Entry) Event {
	return ProcessEntryEvent{
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

func buildImpactEvents(impacts []Impact) []ImpactEvent {
	evts := make([]ImpactEvent, 0, len(impacts))
	for _, i := range impacts {
		new := ImpactEvent{
			Balance:   i.Balance,
			Operation: i.Operation,
			Amount:    i.Amount,
			Rules:     i.Rules,
		}
		evts = append(evts, new)
	}
	return evts
}
