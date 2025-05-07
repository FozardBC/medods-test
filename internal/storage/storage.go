package storage

import (
	"context"
	"errors"
	"medods-test/internal/models"
)

var (
	ErrGuidIsExists = errors.New("GUID is already exists")
)

type Storage interface {
	Close()
	Ping(ctx context.Context) error
	SaveUserInfo(ctx context.Context, UserInfo *models.UserInfo) error
	IsActive(ctx context.Context, guid string) (bool, error)
}
