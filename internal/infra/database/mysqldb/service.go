package mysqldb

import (
	"context"
	"database/sql"

	"github.com/clodoaldomarques/balances-api/internal/domain/accounts"
	"github.com/clodoaldomarques/core-sdk/pkg/zap/logger"
)

var (
	INSERT_ACCOUNT = `insert into accounts (account_id, org_id, limits, balances, created_at, updated_at, status, version) values (?, ?, ?, ?, ?, ?, ?, ?)`
	SELECT_ACCOUNT = `select account_id, org_id, limits, balances, created_at, updated_at, status, version from accounts where account_id = ? and org_id = ?`
	UPDATE_ACCOUNT = `update accounts set limits = ?, balances = ?, updated_at = ?, status = ?, version = ? where account_id = ? and org_id = ?`
	INSERT_ENTRIES = `insert into entries (tracking_id, account_id, org_id, impacts, created_at) values (?, ?, ?, ?, ?)`
)

type Repository struct {
	ctx context.Context
	db  *sql.DB
}

func NewRepository(ctx context.Context) *Repository {
	db, err := Connect()
	if err != nil {
		logger.Error(ctx, "error on connect to database", logger.Fields{"error": err.Error()})
		return &Repository{ctx: ctx}
	}

	return &Repository{ctx: ctx, db: db}
}

func (r Repository) Close() {
	r.db.Close()
}

func (r Repository) SaveNewAccount(ctx context.Context, a accounts.Account) error {
	acc := buildAccountTable(a)
	statement, err := r.db.Prepare(INSERT_ACCOUNT)
	if err != nil {
		logger.Error(ctx, "error on save new account", logger.Fields{"account": a, "error": err.Error(), "sql": INSERT_ACCOUNT})
		return err
	}
	defer statement.Close()
	_, err = statement.Exec(
		acc.AccountID,
		acc.OrgID,
		acc.Limits,
		acc.Balances,
		acc.CreatedAt,
		acc.UpdatedAt,
		acc.Status,
		acc.Version,
	)
	if err != nil {
		return err
	}
	return nil
}
func (r Repository) UpdateExistingAccount(ctx context.Context, a accounts.Account) error {
	acc := buildAccountTable(a)
	statement, err := r.db.Prepare(UPDATE_ACCOUNT)
	if err != nil {
		logger.Error(ctx, "error on update existing account", logger.Fields{"account": a, "error": err.Error(), "sql": UPDATE_ACCOUNT})
		return err
	}
	defer statement.Close()
	_, err = statement.Exec(
		acc.Limits,
		acc.Balances,
		acc.UpdatedAt,
		acc.Status,
		acc.Version,
		acc.AccountID,
		acc.OrgID,
	)
	if err != nil {
		return err
	}
	return nil
}
func (r Repository) RetrieveAccountByID(ctx context.Context, accountID int64, orgID string) (accounts.Account, error) {
	var acc Account
	err := r.db.QueryRow(SELECT_ACCOUNT, accountID, orgID).Scan(
		&acc.AccountID,
		&acc.OrgID,
		&acc.Limits,
		&acc.Balances,
		&acc.CreatedAt,
		&acc.UpdatedAt,
		&acc.Status,
		&acc.Version,
	)
	if err != nil {
		logger.Error(ctx, "error on retrieve existing account", logger.Fields{"account": accountID, "error": err.Error(), "sql": SELECT_ACCOUNT})
		return accounts.Account{}, err
	}
	return acc.toEntity(), nil
}

func (r Repository) SaveEntryAndUpdateAccount(ctx context.Context, e accounts.Entry, a accounts.Account) error {
	tx, err := r.db.BeginTx(r.ctx, nil)
	if err != nil {
		logger.Error(ctx, "error on start new transaction", logger.Fields{"account": a, "entry": e, "error": err.Error()})
		return err
	}
	defer tx.Rollback()

	stat, err := tx.Prepare(INSERT_ENTRIES)
	if err != nil {
		logger.Error(ctx, "error on save new entry", logger.Fields{"account": a, "entry": e, "error": err.Error(), "sql": INSERT_ENTRIES})
		return err
	}
	defer stat.Close()

	en, err := buildEntriesTable(e)
	if err != nil {
		logger.Error(ctx, "error on parse to table", logger.Fields{"entry": e, "error": err})
		return err
	}
	_, err = stat.Exec(
		en.TrackingID,
		en.AccountID,
		en.OrgID,
		en.Impacts,
		en.CreatedAt,
	)
	if err != nil {
		return err
	}

	acc := buildAccountTable(a)
	stat, err = tx.Prepare(UPDATE_ACCOUNT)
	if err != nil {
		logger.Error(ctx, "error on prepare statement", logger.Fields{"account": a, "error": err.Error(), "sql": UPDATE_ACCOUNT})
		return err
	}
	_, err = stat.Exec(
		acc.Limits,
		acc.Balances,
		acc.UpdatedAt,
		acc.Status,
		acc.Version,
		acc.AccountID,
		acc.OrgID,
	)
	if err != nil {
		return err
	}

	tx.Commit()
	return nil
}
