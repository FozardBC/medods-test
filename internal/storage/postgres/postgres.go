package postgres

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"medods-test/internal/models"
	"medods-test/internal/storage"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
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
	IsActivatedColumn   = "is_activated"
)

const (
	BlackListTable  = "blacklist_used_tokens"
	IdRefColumn     = "id_ref_tokens"
	UsedTokenColumn = "used_token"
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

func (s *PostgreStorage) SaveUserInfo(ctx context.Context, UserInfo *models.UserInfo) (int, error) {
	tx, err := s.conn.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		s.log.Error(ErrTxBegin.Error(), "err", err.Error())

		return 0, fmt.Errorf("%w:%w", ErrTxBegin, err)
	}

	tx.Begin(ctx)
	defer tx.Rollback(ctx)

	query := fmt.Sprintf(`
	INSERT INTO %s
	(%s, %s, %s, %s) VALUES ($1, $2, $3, $4) 
	RETURNING id
	`, TokensTable,
		GUIDColumn, RefTokenHashColumn, UserAgentHashColumn, IpHashColumn,
	)

	var id int

	err = s.conn.QueryRow(ctx, query,
		UserInfo.GUID,
		UserInfo.TokenHash,
		UserInfo.UserAgentHash,
		UserInfo.IPhash).Scan(&id)
	if err != nil {
		if pgErr, ok := err.(*pgconn.PgError); ok {
			if pgErr.Code == "23505" {
				s.log.Error(storage.ErrGuidExists.Error(), "err", err.Error())
				s.log.Debug(storage.ErrGuidExists.Error(), "err", err.Error(), "query", query)

				return 0, storage.ErrGuidExists

			}
		}

		s.log.Error(ErrQuery.Error(), "err", err.Error())
		s.log.Debug(ErrQuery.Error(), "err", err.Error(), "query", query)

		return 0, fmt.Errorf("%s:%w", ErrQuery, err)
	}

	err = tx.Commit(ctx)
	if err != nil {
		s.log.Error(ErrTxCommit.Error(), "err", err.Error())
		s.log.Debug(ErrTxCommit.Error(), "err", err.Error(), "query", query)

		return 0, fmt.Errorf("%s:%w", ErrTxCommit, err)
	}

	return id, nil

}

func (s *PostgreStorage) IsActive(ctx context.Context, guid string) (bool, error) {
	tx, err := s.conn.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		s.log.Error(ErrTxBegin.Error(), "err", err.Error())

		return false, fmt.Errorf("%w:%w", ErrTxBegin, err)
	}

	tx.Begin(ctx)
	defer tx.Rollback(ctx)

	var IsActive bool

	query := fmt.Sprintf(`
	SELECT %s FROM %s
	WHERE %s = $1`,
		IsActivatedColumn,
		TokensTable,
		GUIDColumn)

	err = s.conn.QueryRow(ctx, query, guid).Scan(&IsActive)
	if err != nil {
		s.log.Error(ErrQuery.Error(), "err", err.Error())
		s.log.Debug(ErrQuery.Error(), "err", err.Error(), "query", query)

		return false, fmt.Errorf("%s:%w", ErrQuery, err)
	}

	err = tx.Commit(ctx)
	if err != nil {
		s.log.Error(ErrTxCommit.Error(), "err", err.Error())
		s.log.Debug(ErrTxCommit.Error(), "err", err.Error(), "query", query)

		return false, fmt.Errorf("%s:%w", ErrTxCommit, err)
	}
	return IsActive, nil
}

func (s *PostgreStorage) BlockToken(ctx context.Context, hashedToken string, idToken string) error {
	tx, err := s.conn.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		s.log.Error(ErrTxBegin.Error(), "err", err.Error())

		return fmt.Errorf("%w:%w", ErrTxBegin, err)
	}

	query := fmt.Sprintf(`
	INSERT INTO %s (%s,%s)
	VALUES ($1, $2)`,
		BlackListTable,
		IdRefColumn,
		UsedTokenColumn,
	)

	_, err = s.conn.Exec(ctx, query, idToken, hashedToken)
	if err != nil {
		if pgErr, ok := err.(*pgconn.PgError); ok {
			if pgErr.Code == "23505" {
				s.log.Warn(storage.ErrTokenUsedExsits.Error())
				s.log.Debug(storage.ErrGuidExists.Error(), "err", err.Error(), "query", query)
			}
		} else {
			return fmt.Errorf("%w:%w", ErrQuery, err)
		}

	}

	err = tx.Commit(ctx)
	if err != nil {
		s.log.Error(ErrTxCommit.Error(), "err", err.Error())
		s.log.Debug(ErrTxCommit.Error(), "err", err.Error(), "query", query)

		return fmt.Errorf("%s:%w", ErrTxCommit, err)
	}
	return nil

}

func (s *PostgreStorage) IsBlocked(ctx context.Context, hashedToken string) (bool, error) {
	tx, err := s.conn.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		s.log.Error(ErrTxBegin.Error(), "err", err.Error())

		return false, fmt.Errorf("%w:%w", ErrTxBegin, err)
	}

	blocked := false
	var count int

	query := fmt.Sprintf(`
	SELECT COUNT(*) FROM %s
	WHERE %s = $1
	`, BlackListTable,
		UsedTokenColumn,
	)

	err = s.conn.QueryRow(ctx, query, hashedToken).Scan(&count)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			blocked = false
		} else {
			return false, fmt.Errorf("%w:%w", ErrQuery, err)
		}

	}

	if count > 0 {
		blocked = true
	}

	err = tx.Commit(ctx)
	if err != nil {
		s.log.Error(ErrTxCommit.Error(), "err", err.Error())
		s.log.Debug(ErrTxCommit.Error(), "err", err.Error(), "query", query)

		return false, fmt.Errorf("%s:%w", ErrTxCommit, err)
	}
	return blocked, nil
}

func (s *PostgreStorage) FindByGUID(ctx context.Context, guid string) (*models.UserInfo, int, error) {
	tx, err := s.conn.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		s.log.Error(ErrTxBegin.Error(), "err", err.Error())

		return nil, 0, fmt.Errorf("%w:%w", ErrTxBegin, err)
	}

	var UserInfo models.UserInfo
	var id int

	query := fmt.Sprintf(`
	SELECT %s, %s,%s,%s,%s FROM %s
	WHERE %s = $1
	`, IdColumn, GUIDColumn, RefTokenHashColumn, UserAgentHashColumn, IpHashColumn,
		TokensTable,
		GUIDColumn,
	)

	err = s.conn.QueryRow(ctx, query, guid).Scan(
		&id,
		&UserInfo.GUID,
		&UserInfo.TokenHash,
		&UserInfo.UserAgentHash,
		&UserInfo.IPhash,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			s.log.Error("GUID not found")

			return nil, 0, fmt.Errorf("GUID not found:%w", err)
		} else {
			s.log.Error(ErrQuery.Error(), "error", err.Error())
			return nil, 0, fmt.Errorf("%w:%w", ErrQuery, err)
		}

	}

	err = tx.Commit(ctx)
	if err != nil {
		s.log.Error(ErrTxCommit.Error(), "err", err.Error())
		s.log.Debug(ErrTxCommit.Error(), "err", err.Error(), "query", query)

		return nil, 0, fmt.Errorf("%s:%w", ErrTxCommit, err)
	}

	return &UserInfo, id, nil
}

func (s *PostgreStorage) Logout(ctx context.Context, guid string) error {
	tx, err := s.conn.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		s.log.Error(ErrTxBegin.Error(), "err", err.Error())

		return fmt.Errorf("%w:%w", ErrTxBegin, err)
	}

	query := fmt.Sprintf(`
	UPDATE %s
	SET %s = FALSE
	WHERE %s = $1`,
		TokensTable,
		IsActivatedColumn,
		GUIDColumn,
	)

	_, err = s.conn.Exec(ctx, query, guid)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			s.log.Error("GUID not found")

			return fmt.Errorf("GUID not found:%w", err)
		} else {
			s.log.Error(ErrQuery.Error(), "error", err.Error())
			return fmt.Errorf("%w:%w", ErrQuery, err)
		}

	}

	err = tx.Commit(ctx)
	if err != nil {
		s.log.Error(ErrTxCommit.Error(), "err", err.Error())
		s.log.Debug(ErrTxCommit.Error(), "err", err.Error(), "query", query)

		return fmt.Errorf("%s:%w", ErrTxCommit, err)
	}

	return nil
}

func (s *PostgreStorage) UpdateUserInfo(ctx context.Context, UserInfo *models.UserInfo) error {
	tx, err := s.conn.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		s.log.Error(ErrTxBegin.Error(), "err", err.Error())

		return fmt.Errorf("%w:%w", ErrTxBegin, err)
	}

	tx.Begin(ctx)
	defer tx.Rollback(ctx)

	query := fmt.Sprintf(`
	UPDATE %s
	SET %s = $1, %s = $2, %s = $3
	WHERE %s = $4
	`, TokensTable,
		RefTokenHashColumn, UserAgentHashColumn, IpHashColumn,
		GUIDColumn,
	)

	_, err = s.conn.Exec(ctx, query,
		UserInfo.TokenHash,
		UserInfo.UserAgentHash,
		UserInfo.IPhash,
		UserInfo.GUID)
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
