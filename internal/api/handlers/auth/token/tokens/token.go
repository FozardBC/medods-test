package tokens

import (
	"context"
	"encoding/base64"
	"errors"
	"log/slog"
	"medods-test/internal/lib/api/response"
	"medods-test/internal/lib/jwt"
	"medods-test/internal/models"
	"medods-test/internal/storage"
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

type Response struct {
	Resp        response.Response `json:"response"`
	AccessToken string            `json:"accessToken"`
}

var (
	liveAccess  = time.Hour * 24     // 1 day
	liveRefresh = time.Hour * 24 * 7 // 1 week
)

type Saver interface {
	SaveUserInfo(ctx context.Context, UserInfo *models.UserInfo) (int, error)
}

// @Summary Создание новых токенов
// @Description Генерирует новую пару access и refresh токенов для пользователя
// @Tags Auth
// @Accept json
// @Produce json
// @Param request body Request true "Данные для генерации токенов"
// @Success 200 {object} Response "Успешная генерация токенов"
// @Success 200 {string} string "Set-Cookie: refreshToken={token}; Path=/; Domain=localhost; Max-Age={liveRefresh}; HttpOnly"
// @Failure 400 {object} response.Response "Невалидные входные данные"
// @Failure 500 {object} response.Response "Внутренняя ошибка сервера"
// @Router /auth/tokens [post]
func New(log *slog.Logger, saver Saver) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()

		logHandler := log.With(
			"requestID", requestid.Get(c),
		)

		var req Request

		if c.Request.UserAgent() == "" {
			logHandler.Error("UserAgent from request is empty")

			c.JSON(http.StatusBadRequest, response.Error("Bad request"))
			return
		}

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

		accToken, err := jwt.NewAccessToken(req.GUID, liveAccess)
		if err != nil {
			logHandler.Error("failed to generate jwt", "error", err.Error())

			logHandler.Debug("debug", "guid", req.GUID)

			c.JSON(http.StatusInternalServerError, response.Error("Internal error"))
			return
		}

		refToken, err := jwt.NewRefreshToken(req.GUID, liveRefresh)
		if err != nil {
			logHandler.Error("failed to generate jwt", "error", err.Error())

			logHandler.Debug("debug", "guid", req.GUID)

			c.JSON(http.StatusInternalServerError, response.Error("Internal error"))
			return
		}

		hashedRefToken, err := jwt.HashJWTbcrypt(refToken) // рефреш токен сначала сжимается до 64 байт чтобы затем захешировать его в bcrypt (ограничение 72 байта)
		if err != nil {
			logHandler.Error("failed to hash refresh token", "error", err.Error())

			logHandler.Debug("debug", "guid", req.GUID)

			c.JSON(http.StatusInternalServerError, response.Error("Internal error"))
			return
		}

		hashedUserAgent, err := bcrypt.GenerateFromPassword([]byte(c.Request.UserAgent()), bcrypt.DefaultCost)
		if err != nil {
			logHandler.Error("failed to hash user agent", "error", err.Error())

			logHandler.Debug("debug", "guid", req.GUID)

			c.JSON(http.StatusInternalServerError, response.Error("Internal error"))
			return
		}

		hashedIP, err := bcrypt.GenerateFromPassword([]byte(c.ClientIP()), bcrypt.DefaultCost)
		if err != nil {
			logHandler.Error("failed to hash user agent", "error", err.Error())

			logHandler.Debug("debug", "guid", req.GUID)

			c.JSON(http.StatusInternalServerError, response.Error("Internal error"))
			return
		}

		UserInfo := &models.UserInfo{
			GUID:          req.GUID,
			UserAgentHash: string(hashedUserAgent),
			IPhash:        string(hashedIP),
			TokenHash:     hashedRefToken,
		}

		if _, err := saver.SaveUserInfo(ctx, UserInfo); err != nil {
			if errors.Is(err, storage.ErrGuidExists) {
				logHandler.Error(err.Error())

				logHandler.Debug("debug", "guid", req.GUID)

				c.JSON(http.StatusBadRequest, response.Error("GUID is already exists"))

				return

			}
			logHandler.Error("failed to save hash User Info", "error", err.Error())

			logHandler.Debug("debug", "guid", req.GUID)

			c.JSON(http.StatusInternalServerError, response.Error("Internal error"))
			return
		}

		c.SetCookie(
			"refreshToken", base64.StdEncoding.EncodeToString([]byte(refToken)),
			int(liveRefresh.Seconds()),
			"/",
			"localhost",
			false,
			true,
		)

		c.JSON(http.StatusOK, Response{Resp: response.OK(), AccessToken: accToken})

	}
}
