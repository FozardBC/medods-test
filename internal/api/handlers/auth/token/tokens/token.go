package tokens

import (
	"context"
	"crypto/sha512"
	"encoding/base64"
	"log/slog"
	"medods-test/internal/lib/api/response"
	"medods-test/internal/lib/jwt"
	"net/http"
	"time"

	"github.com/gin-contrib/requestid"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"golang.org/x/crypto/bcrypt"
)

type Request struct {
	GUID string `json:"guid" validate:"required,uuid"`
}

var (
	liveAccess  = time.Hour * 24     // 1 day
	liveRefresh = time.Hour * 24 * 7 // 1 week
)

type TokenSaver interface {
	SaveToken(ctx context.Context, token string) error
}

func New(log *slog.Logger, saver TokenSaver) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()

		logHandler := log.With(
			"requestID", requestid.Get(c),
		)

		var req Request

		if err := c.BindJSON(&req); err != nil {
			logHandler.Error("failed to decode request body", "error", err.Error())

			c.JSON(http.StatusBadRequest, response.Error("failed decode body"))
			return
		}

		if err := validator.New().Struct(req); err != nil {
			validatorErr := err.(validator.ValidationErrors)

			logHandler.Error("invalid request", "err", err.Error())

			c.JSON(http.StatusBadRequest, response.ValidationError(validatorErr))

			return
		}

		accToken, err := jwt.NewToken(req.GUID, liveAccess)
		if err != nil {
			logHandler.Error("failed to generate jwt", "error", err.Error())

			logHandler.Debug("debug", "guid", req.GUID)

			c.JSON(http.StatusInternalServerError, response.Error("Internal error"))
			return
		}

		refToken, err := jwt.NewToken(req.GUID, liveRefresh)
		if err != nil {
			logHandler.Error("failed to generate jwt", "error", err.Error())

			logHandler.Debug("debug", "guid", req.GUID)

			c.JSON(http.StatusInternalServerError, response.Error("Internal error"))
			return
		}

		hashedRefToken, err := HashJWTbcrypt(refToken) // рефреш токен сначала сжимается до 64 байт чтобы затем захешировать его в bcrypt (ограничение 72 байта)
		if err != nil {
			logHandler.Error("failed to hash refresh token", "error", err.Error())

			logHandler.Debug("debug", "guid", req.GUID)

			c.JSON(http.StatusInternalServerError, response.Error("Internal error"))
			return
		}

		if err := saver.SaveToken(ctx, string(hashedRefToken)); err != nil {
			logHandler.Error("failed to save hash refresh token", "error", err.Error())

			logHandler.Debug("debug", "guid", req.GUID)

			c.JSON(http.StatusInternalServerError, response.Error("Internal error"))
			return
		}

		c.SetCookie(
			"accessToken", base64.URLEncoding.EncodeToString([]byte(accToken)),
			int(liveAccess.Seconds()),
			"/",
			"localhost",
			false,
			true,
		)

		c.SetCookie(
			"refreshToken", base64.URLEncoding.EncodeToString([]byte(accToken)),
			int(liveRefresh.Seconds()),
			"/",
			"localhost",
			false,
			true,
		)

		c.JSON(http.StatusOK, response.OK())

	}
}

// чтобы захешировать jwt refresh token в bcrypt, необходимо его сжать до 64 байт
func HashJWTbcrypt(jwt string) (string, error) {
	shaHash := sha512.Sum512([]byte(jwt))
	shaHashStr := string(shaHash[:])

	hashedJwtToken, err := bcrypt.GenerateFromPassword([]byte(shaHashStr), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hashedJwtToken), nil
}
