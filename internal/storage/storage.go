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
	SaveUserInfo(ctx context.Context, UserInfo *models.UserInfo) (int, error)
	UpdateUserInfo(ctx context.Context, UserInfo *models.UserInfo) error
	IsActive(ctx context.Context, guid string) (bool, error)
	Logout(ctx context.Context, guid string) error
	BlockToken(ctx context.Context, hashedToken string, idToken string) error
	IsBlocked(ctx context.Context, hashedToken string) (bool, error)
	FindByGUID(ctx context.Context, guid string) (*models.UserInfo, int, error)
}
