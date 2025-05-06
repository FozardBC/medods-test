package storage

import "context"

type Storage interface {
	Close()
	Ping(ctx context.Context) error
	SaveToken(ctx context.Context, token string) error
}
