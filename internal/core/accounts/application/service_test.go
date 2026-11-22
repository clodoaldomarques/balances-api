package application

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/clodoaldomarques/balances-api/internal/core/accounts/domain"
	"github.com/clodoaldomarques/balances-api/internal/core/accounts/ports"
	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

func TestNewService(t *testing.T) {
	tests := []struct {
		name  string
		setup func(ctrl *gomock.Controller) (ports.AccountRepository, ports.AccountEventPublisher)
		want  func(t *testing.T, s *Service)
	}{
		{
			name: "when create new service with success",
			setup: func(ctrl *gomock.Controller) (ports.AccountRepository, ports.AccountEventPublisher) {
				return ports.NewMockAccountRepository(ctrl), ports.NewMockAccountEventPublisher(ctrl)
			},
			want: func(t *testing.T, s *Service) {
				assert.NotEmpty(t, s)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			r, p := tt.setup(ctrl)
			tt.want(t, NewService(r, p))
		})
	}
}

func TestService_CreateNewAccount(t *testing.T) {
	tests := []struct {
		name  string
		setup func(ctrl *gomock.Controller) *Service
		args  func() domain.Account
		want  func(t *testing.T, acc domain.Account, e error)
	}{
		{
			name: "when create account with sucess",
			setup: func(ctrl *gomock.Controller) *Service {
				rep := ports.NewMockAccountRepository(ctrl)
				rep.EXPECT().SaveNewAccount(gomock.Any(), gomock.Any()).Return(nil).Times(1)
				pub := ports.NewMockAccountEventPublisher(ctrl)
				pub.EXPECT().Emit(gomock.Any(), gomock.Any()).Times(1)
				return NewService(rep, pub)
			},
			args: func() domain.Account {
				return buildAccount()
			},
			want: func(t *testing.T, acc domain.Account, e error) {
				assert.Nil(t, e)
			},
		},
		{
			name: "when repository send an error",
			setup: func(ctrl *gomock.Controller) *Service {
				rep := ports.NewMockAccountRepository(ctrl)
				rep.EXPECT().SaveNewAccount(gomock.Any(), gomock.Any()).Return(errors.New("repository not found")).Times(1)
				pub := ports.NewMockAccountEventPublisher(ctrl)
				return NewService(rep, pub)
			},
			args: func() domain.Account {
				return buildAccount()
			},
			want: func(t *testing.T, acc domain.Account, e error) {
				assert.NotNil(t, e)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			s := tt.setup(ctrl)
			acc, e := s.CreateNewAccount(context.Background(), tt.args())

			tt.want(t, acc, e)
		})
	}
}

func TestService_UpdateAccountLimits(t *testing.T) {
	tests := []struct {
		name  string
		setup func(ctrl *gomock.Controller) *Service
		args  func() (int64, string, map[string]decimal.Decimal)
		want  func(t *testing.T, acc domain.Account, e error)
	}{
		{
			name: "when update a limit with sucess",
			setup: func(ctrl *gomock.Controller) *Service {
				rep := ports.NewMockAccountRepository(ctrl)
				acc := buildAccount()
				rep.EXPECT().RetrieveAccountByID(gomock.Any(), gomock.Eq(int64(230513)), gomock.Eq("TN-12345678")).Return(acc, nil).Times(1)
				rep.EXPECT().UpdateExistingAccount(gomock.Any(), gomock.Any()).Return(nil).Times(1)

				pub := ports.NewMockAccountEventPublisher(ctrl)
				pub.EXPECT().Emit(gomock.Any(), gomock.Any()).AnyTimes()
				return NewService(rep, pub)
			},
			args: func() (int64, string, map[string]decimal.Decimal) {
				return int64(230513), "TN-12345678", map[string]decimal.Decimal{
					domain.MaxLimit: decimal.NewFromInt(150),
				}
			},
			want: func(t *testing.T, acc domain.Account, e error) {
				assert.Nil(t, e)
			},
		},
		{
			name: "when account not found",
			setup: func(ctrl *gomock.Controller) *Service {
				rep := ports.NewMockAccountRepository(ctrl)
				acc := domain.Account{}
				rep.EXPECT().RetrieveAccountByID(gomock.Any(), gomock.Eq(int64(230513)), gomock.Eq("TN-12345678")).Return(acc, errors.New("domain.Account not found")).Times(1)

				pub := ports.NewMockAccountEventPublisher(ctrl)
				pub.EXPECT().Emit(gomock.Any(), gomock.Any()).AnyTimes()
				return NewService(rep, pub)
			},
			args: func() (int64, string, map[string]decimal.Decimal) {
				return int64(230513), "TN-12345678", map[string]decimal.Decimal{
					domain.MaxLimit: decimal.NewFromInt(150),
				}
			},
			want: func(t *testing.T, acc domain.Account, e error) {
				assert.NotNil(t, e)
				assert.Equal(t, "domain.Account not found", e.Error())
			},
		},
		{
			name: "when send invalid limit",
			setup: func(ctrl *gomock.Controller) *Service {
				rep := ports.NewMockAccountRepository(ctrl)
				acc := buildAccount()
				rep.EXPECT().RetrieveAccountByID(gomock.Any(), gomock.Eq(int64(230513)), gomock.Eq("TN-12345678")).Return(acc, nil).Times(1)

				pub := ports.NewMockAccountEventPublisher(ctrl)
				pub.EXPECT().Emit(gomock.Any(), gomock.Any()).AnyTimes()
				return NewService(rep, pub)
			},
			args: func() (int64, string, map[string]decimal.Decimal) {
				return int64(230513), "TN-12345678", map[string]decimal.Decimal{
					domain.TotalLimit: decimal.NewFromInt(150),
				}
			},
			want: func(t *testing.T, acc domain.Account, e error) {
				assert.NotNil(t, e)
				assert.Equal(t, "new limit can not great than max limit", e.Error())
			},
		},
		{
			name: "when send an error on update account",
			setup: func(ctrl *gomock.Controller) *Service {
				rep := ports.NewMockAccountRepository(ctrl)
				acc := buildAccount()
				rep.EXPECT().RetrieveAccountByID(gomock.Any(), gomock.Eq(int64(230513)), gomock.Eq("TN-12345678")).Return(acc, nil).Times(1)
				rep.EXPECT().UpdateExistingAccount(gomock.Any(), gomock.Any()).Return(errors.New("error on update account")).Times(1)

				pub := ports.NewMockAccountEventPublisher(ctrl)
				pub.EXPECT().Emit(gomock.Any(), gomock.Any()).AnyTimes()
				return NewService(rep, pub)
			},
			args: func() (int64, string, map[string]decimal.Decimal) {
				return int64(230513), "TN-12345678", map[string]decimal.Decimal{
					domain.MaxLimit: decimal.NewFromInt(150),
				}
			},
			want: func(t *testing.T, acc domain.Account, e error) {
				assert.NotNil(t, e)
				assert.Equal(t, "error on update account", e.Error())
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			s := tt.setup(ctrl)
			accID, orgID, limits := tt.args()
			acc, e := s.UpdateAccountLimits(context.Background(), accID, orgID, limits)
			tt.want(t, acc, e)
		})
	}
}

func TestService_UpdateAccountStatus(t *testing.T) {
	tests := []struct {
		name  string
		setup func(ctrl *gomock.Controller) *Service
		args  func() (int64, string, domain.Status)
		want  func(t *testing.T, acc domain.Account, e error)
	}{
		{
			name: "when disable account with sucess",
			setup: func(ctrl *gomock.Controller) *Service {
				rep := ports.NewMockAccountRepository(ctrl)
				acc := buildAccount()
				rep.EXPECT().RetrieveAccountByID(gomock.Any(), gomock.Eq(int64(230513)), gomock.Eq("TN-12345678")).Return(acc, nil).Times(1)
				rep.EXPECT().UpdateExistingAccount(gomock.Any(), gomock.Any()).Return(nil).Times(1)

				pub := ports.NewMockAccountEventPublisher(ctrl)
				pub.EXPECT().Emit(gomock.Any(), gomock.Any()).Times(1)
				return NewService(rep, pub)
			},
			args: func() (int64, string, domain.Status) {
				return int64(230513), "TN-12345678", domain.Inative
			},
			want: func(t *testing.T, acc domain.Account, e error) {
				assert.Nil(t, e)
			},
		},
		{
			name: "when account not found",
			setup: func(ctrl *gomock.Controller) *Service {
				rep := ports.NewMockAccountRepository(ctrl)
				acc := domain.Account{}
				rep.EXPECT().RetrieveAccountByID(gomock.Any(), gomock.Eq(int64(230513)), gomock.Eq("TN-12345678")).Return(acc, errors.New("account not found")).Times(1)
				pub := ports.NewMockAccountEventPublisher(ctrl)
				return NewService(rep, pub)
			},
			args: func() (int64, string, domain.Status) {
				return int64(230513), "TN-12345678", domain.Inative
			},
			want: func(t *testing.T, acc domain.Account, e error) {
				assert.NotNil(t, e)
				assert.Equal(t, "account not found", e.Error())
			},
		},
		{
			name: "when account update failed",
			setup: func(ctrl *gomock.Controller) *Service {
				rep := ports.NewMockAccountRepository(ctrl)
				acc := buildAccount()
				rep.EXPECT().RetrieveAccountByID(gomock.Any(), gomock.Eq(int64(230513)), gomock.Eq("TN-12345678")).Return(acc, nil).Times(1)
				rep.EXPECT().UpdateExistingAccount(gomock.Any(), gomock.Any()).Return(errors.New("account update failed")).Times(1)
				pub := ports.NewMockAccountEventPublisher(ctrl)
				return NewService(rep, pub)
			},
			args: func() (int64, string, domain.Status) {
				return int64(230513), "TN-12345678", domain.Inative
			},
			want: func(t *testing.T, acc domain.Account, e error) {
				assert.NotNil(t, e)
				assert.Equal(t, "account update failed", e.Error())
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			s := tt.setup(ctrl)
			accID, orgID, status := tt.args()
			acc, e := s.UpdateAccountStatus(context.Background(), accID, orgID, status)
			tt.want(t, acc, e)
		})
	}
}

func TestService_ProcessEntry(t *testing.T) {
	tests := []struct {
		name  string
		setup func(ctrl *gomock.Controller) *Service
		args  func() domain.Entry
		want  func(t *testing.T, acc domain.Account, e error)
	}{
		{
			name: "when process credit entry with sucess",
			setup: func(ctrl *gomock.Controller) *Service {
				rep := ports.NewMockAccountRepository(ctrl)
				acc := buildAccount()
				rep.EXPECT().RetrieveAccountByID(gomock.Any(), gomock.Eq(int64(230513)), gomock.Eq("TN-12345678")).Return(acc, nil).Times(1)
				rep.EXPECT().SaveEntryAndUpdateAccount(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).Times(1)
				pub := ports.NewMockAccountEventPublisher(ctrl)
				pub.EXPECT().Emit(gomock.Any(), gomock.Any()).AnyTimes()
				return NewService(rep, pub)
			},
			args: func() domain.Entry {
				return buildEntry(decimal.NewFromInt(10))
			},
			want: func(t *testing.T, acc domain.Account, e error) {
				assert.Nil(t, e)
			},
		},
		{
			name: "when account not found",
			setup: func(ctrl *gomock.Controller) *Service {
				rep := ports.NewMockAccountRepository(ctrl)
				rep.EXPECT().RetrieveAccountByID(gomock.Any(), gomock.Eq(int64(230513)), gomock.Eq("TN-12345678")).Return(domain.Account{}, errors.New("account not found")).Times(1)
				pub := ports.NewMockAccountEventPublisher(ctrl)
				pub.EXPECT().Emit(gomock.Any(), gomock.Any()).AnyTimes()
				return NewService(rep, pub)
			},
			args: func() domain.Entry {
				return buildEntry(decimal.NewFromInt(10))
			},
			want: func(t *testing.T, acc domain.Account, e error) {
				assert.NotNil(t, e)
				assert.Equal(t, "account not found", e.Error())
			},
		},
		{
			name: "when error on update account",
			setup: func(ctrl *gomock.Controller) *Service {
				rep := ports.NewMockAccountRepository(ctrl)
				acc := buildAccount()
				rep.EXPECT().RetrieveAccountByID(gomock.Any(), gomock.Eq(int64(230513)), gomock.Eq("TN-12345678")).Return(acc, nil).Times(1)
				rep.EXPECT().SaveEntryAndUpdateAccount(gomock.Any(), gomock.Any(), gomock.Any()).Return(errors.New("illegal operation")).Times(1)
				pub := ports.NewMockAccountEventPublisher(ctrl)
				pub.EXPECT().Emit(gomock.Any(), gomock.Any()).AnyTimes()
				return NewService(rep, pub)
			},
			args: func() domain.Entry {
				return buildEntry(decimal.NewFromInt(10))
			},
			want: func(t *testing.T, acc domain.Account, e error) {
				assert.NotNil(t, e)
				assert.Equal(t, "illegal operation", e.Error())
			},
		},
		{
			name: "when error on insuficient balance",
			setup: func(ctrl *gomock.Controller) *Service {
				rep := ports.NewMockAccountRepository(ctrl)
				acc := buildAccount()
				rep.EXPECT().RetrieveAccountByID(gomock.Any(), gomock.Eq(int64(230513)), gomock.Eq("TN-12345678")).Return(acc, nil).Times(1)
				pub := ports.NewMockAccountEventPublisher(ctrl)
				pub.EXPECT().Emit(gomock.Any(), gomock.Any()).AnyTimes()
				return NewService(rep, pub)
			},
			args: func() domain.Entry {
				return buildEntry(decimal.NewFromInt(1000))
			},
			want: func(t *testing.T, acc domain.Account, e error) {
				assert.NotNil(t, e)
				assert.Equal(t, "insuficient balance", e.Error())
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			s := tt.setup(ctrl)
			acc, e := s.ProcessEntry(context.Background(), tt.args())
			tt.want(t, acc, e)
		})
	}
}

func buildEntry(amount decimal.Decimal) domain.Entry {
	return domain.Entry{
		TrackingID: uuid.NewString(),
		AccountID:  int64(230513),
		OrgID:      "TN-12345678",
		Impacts: []domain.Impact{
			{
				Balance:   "available_balance",
				Operation: "DEBIT",
				Amount:    amount,
				Rules:     []string{domain.ConsiderAvailableBalance},
			},
			{
				Balance:   "savings_balance",
				Operation: "DEBIT",
				Amount:    amount,
				Rules:     []string{domain.ConsiderAvailableSavings},
			},
			{
				Balance:   "blocked_balance",
				Operation: "DEBIT",
				Amount:    amount,
				Rules:     []string{domain.ConsiderBlockedBalance},
			},
		},
		CreatedAt: time.Now(),
	}
}

func buildAccount() domain.Account {
	return domain.Account{
		AccountID: int64(23052013),
		OrgID:     "TN-12345678",
		Limits: map[string]decimal.Decimal{
			domain.MaxLimit:       decimal.NewFromInt(100),
			domain.TotalLimit:     decimal.NewFromInt(100),
			domain.OverdraftLimit: decimal.NewFromInt(50),
		},
		Balances: map[string]decimal.Decimal{
			domain.Available: decimal.NewFromInt(100),
			domain.Savings:   decimal.NewFromInt(100),
			domain.Blocked:   decimal.NewFromInt(100),
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Status:    domain.Active,
		Version:   1,
	}
}
