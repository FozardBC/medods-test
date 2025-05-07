package storage

import (
	"context"
	"errors"
	"medods-test/internal/models"
)

var (
	ErrGuidExists      = errors.New("GUID is already exists")
	ErrTokenUsedExsits = errors.New("token is alredy exists")
)

type Storage interface {
	Close()
	Ping(ctx context.Context) error
	SaveUserInfo(ctx context.Context, UserInfo *models.UserInfo) error
	IsActive(ctx context.Context, guid string) (bool, error)
	BlockToken(ctx context.Context, hashedToken string, idToken string) error
}
