package storage

import "context"

type Storage interface {
	Close()
	Ping(ctx context.Context) error
}
