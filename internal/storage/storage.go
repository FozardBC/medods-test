package storage

import (
	"context"
	"medods-test/internal/models"
)

type Storage interface {
	Close()
	Ping(ctx context.Context) error
	SaveUserInfo(ctx context.Context, UserInfo *models.UserInfo) error
}
