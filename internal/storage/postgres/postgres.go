package postgres

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"medods-test/internal/models"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	TokensTable         = "ref_tokens"
	IdColumn            = "id"
	GUIDColumn          = "guid"
	RefTokenHashColumn  = "token_hash"
	UserAgentHashColumn = "user_agent_hash"
	IpHashColumn        = "ip_hash"
	CreatedColumn       = "created_at"
	UpdatedColum        = "updated_at"
)

var (
	ErrConnectString = errors.New("can't connect to Postgres")
	ErrTxBegin       = errors.New("can't start transaction")
	ErrTxCommit      = errors.New("can't commit transaction")
	ErrQuery         = errors.New("can't do query")
)

type PostgreStorage struct {
	conn *pgxpool.Pool
	log  *slog.Logger
}

func New(ctx context.Context, log *slog.Logger, connString string) (*PostgreStorage, error) {
	log.Debug("Connecting to database", "Connect String", connString)

	conn, err := pgxpool.New(ctx, connString)
	if err != nil {
		log.Error(ErrConnectString.Error(), "err", err.Error())

		return nil, fmt.Errorf("%w:%w", ErrConnectString, err)
	}

	err = conn.Ping(ctx)
	if err != nil {
		log.Error(ErrConnectString.Error(), "err", err.Error())

		return nil, fmt.Errorf("%w:%w", ErrConnectString, err)
	}

	log.Debug("Database is connected")

	return &PostgreStorage{
		conn: conn,
		log:  log,
	}, nil
}

func (s *PostgreStorage) Close() {
	s.conn.Close()
}

func (s *PostgreStorage) Ping(ctx context.Context) error {
	ctxTimeOut, cancel := context.WithTimeout(ctx, time.Second*10)
	defer cancel()

	err := s.conn.Ping(ctxTimeOut)
	if err != nil {
		return fmt.Errorf("no database connection:%w", err)
	}

	return nil
}

func (s *PostgreStorage) SaveUserInfo(ctx context.Context, UserInfo *models.UserInfo) error {
	tx, err := s.conn.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		s.log.Error(ErrTxBegin.Error(), "err", err.Error())

		return fmt.Errorf("%w:%w", ErrTxBegin, err)
	}

	tx.Begin(ctx)
	defer tx.Rollback(ctx)

	query := fmt.Sprintf(`
	INSERT INTO %s
	(%s, %s, %s, %s) VALUES ($1, $2, $3, $4) 
	`, TokensTable,
		GUIDColumn, RefTokenHashColumn, UserAgentHashColumn, IpHashColumn,
	)

	_, err = s.conn.Exec(ctx, query,
		UserInfo.GUID,
		UserInfo.TokenHash,
		UserInfo.UserAgentHash,
		UserInfo.IPhash)
	if err != nil {
		s.log.Error(ErrQuery.Error(), "err", err.Error())
		s.log.Debug(ErrQuery.Error(), "err", err.Error(), "query", query)

		return fmt.Errorf("%s:%w", ErrQuery, err)
	}

	err = tx.Commit(ctx)
	if err != nil {
		s.log.Error(ErrTxCommit.Error(), "err", err.Error())
		s.log.Debug(ErrTxCommit.Error(), "err", err.Error(), "query", query)

		return fmt.Errorf("%s:%w", ErrTxCommit, err)
	}

	return nil

}

func (s *PostgreStorage) SaveUserAgentHash(ctx context.Context, UserAgentHash string) error {
	tx, err := s.conn.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		s.log.Error(ErrTxBegin.Error(), "err", err.Error())

		return fmt.Errorf("%w:%w", ErrTxBegin, err)
	}

	tx.Begin(ctx)
	defer tx.Rollback(ctx)

	query := fmt.Sprintf(`
	INSERT INTO %s
	(%s) VALUES ($1) 
	`, TokensTable,
		UserAgentHash,
	)

	_, err = s.conn.Exec(ctx, query, UserAgentHash)
	if err != nil {
		s.log.Error(ErrQuery.Error(), "err", err.Error())
		s.log.Debug(ErrQuery.Error(), "err", err.Error(), "query", query)

		return fmt.Errorf("%s:%w", ErrQuery, err)
	}

	err = tx.Commit(ctx)
	if err != nil {
		s.log.Error(ErrTxCommit.Error(), "err", err.Error())
		s.log.Debug(ErrTxCommit.Error(), "err", err.Error(), "query", query)

		return fmt.Errorf("%s:%w", ErrTxCommit, err)
	}

	return nil
}

func (s *PostgreStorage) SaveIpHash(ctx context.Context, ipHash string) error {
	tx, err := s.conn.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		s.log.Error(ErrTxBegin.Error(), "err", err.Error())

		return fmt.Errorf("%w:%w", ErrTxBegin, err)
	}

	tx.Begin(ctx)
	defer tx.Rollback(ctx)

	query := fmt.Sprintf(`
	INSERT INTO %s
	(%s) VALUES ($1) 
	`, TokensTable,
		IpHashColumn,
	)

	_, err = s.conn.Exec(ctx, query, ipHash)
	if err != nil {
		s.log.Error(ErrQuery.Error(), "err", err.Error())
		s.log.Debug(ErrQuery.Error(), "err", err.Error(), "query", query)

		return fmt.Errorf("%s:%w", ErrQuery, err)
	}

	err = tx.Commit(ctx)
	if err != nil {
		s.log.Error(ErrTxCommit.Error(), "err", err.Error())
		s.log.Debug(ErrTxCommit.Error(), "err", err.Error(), "query", query)

		return fmt.Errorf("%s:%w", ErrTxCommit, err)
	}

	return nil
}
