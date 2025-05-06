package postgres

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
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

func (s *PostgreStorage) SaveToken(ctx context.Context, token string) error {
	return nil
}
