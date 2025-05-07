package refresh

import (
	"context"
	"encoding/base64"
	"errors"
	"log/slog"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"medods-test/internal/lib/api/response"
	libJwt "medods-test/internal/lib/jwt"
	"medods-test/internal/models"
	"medods-test/internal/storage"

	"github.com/gin-contrib/requestid"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt"
	"golang.org/x/crypto/bcrypt"
)

type Response struct {
	Resp        response.Response `json:"response"`
	AccessToken string            `json:"accessToken"`
}

var (
	liveAccess  = time.Hour * 24     // 1 day
	liveRefresh = time.Hour * 24 * 7 // 1 week
)

type Storage interface {
	FindByGUID(ctx context.Context, guid string) (*models.UserInfo, int, error)
	UpdateUserInfo(ctx context.Context, UserInfo *models.UserInfo) error
	BlockToken(ctx context.Context, hashedToken string, idToken string) error
	IsBlocked(ctx context.Context, hashedToken string) (bool, error)
}

// RefreshToken godoc
// @Summary Обновление пары JWT токенов
// @Description Проверяет валидность access и refresh токенов, их соответствие, отсутствие в черном списке. Выдает новую пару токенов, добавляет старые в черный список и обновляет данные пользователя.
// @Description Refresh token читается из cookie "Cookie:refreshToken="
// @Tags Refresh tokens
// @Accept json
// @Produce json
// @Param Authorization header string true "Access токен в формате 'Bearer <token>'"
// @Success 200 {object} Response "Успешное обновление токенов"
// @Failure 400 {object} response.Response "Некорректный запрос (например, GUID уже существует)"
// @Failure 401 {string} string "Неавторизован (невалидные токены, токены в черном списке и т.д.)"
// @Failure 500 {object} response.Response "Внутренняя ошибка сервера"
// @Router /refresh [post]
func New(log *slog.Logger, storager Storage) gin.HandlerFunc {
	return func(c *gin.Context) {
		// проверить не в блек листе ли Рефреш токен
		ctx := c.Request.Context()

		logHandler := log.With(
			"requestID", requestid.Get(c),
		)

		//получили и проверили подпись токена
		authHeader := c.Request.Header.Get("Authorization")
		if authHeader == "" {
			logHandler.Error("failed to get token from headers")
			c.JSON(http.StatusUnauthorized, "Unauthorized")

			return
		}

		authTokens := strings.Split(authHeader, " ")

		if len(authTokens) != 2 || authTokens[0] != "Bearer" {
			logHandler.Error("failed to get beraer")
			c.JSON(http.StatusUnauthorized, "Unauthorized")
		}

		blocked, err := storager.IsBlocked(ctx, authTokens[1])
		if err != nil {
			logHandler.Error("failed to check blocked token", "error", err)

			c.JSON(http.StatusUnauthorized, "Unauthorized")
			return
		}

		if blocked {
			logHandler.Error("token in blacklist", "error", err)

			c.JSON(http.StatusUnauthorized, "Unauthorized")
			return
		}

		accessToken, err := libJwt.VerifyToken(authTokens[1], "access")
		if err != nil {
			logHandler.Error("failed to verify access token")
			c.JSON(http.StatusUnauthorized, "Unauthorized")
			return
		}

		AccessClaims := accessToken.Claims

		jwtClaims, ok := AccessClaims.(jwt.MapClaims)
		if !ok {
			logHandler.Error("unexepted type of claims")

			c.JSON(http.StatusUnauthorized, "Unauthorized")
			return
		}

		guidAccess := jwtClaims["guid"].(string)
		createdAtAccess := jwtClaims["created_at"].(float64)

		EncodedRefreshToken, err := c.Cookie("refreshToken")
		if err != nil {
			logHandler.Error("failed to get refresh token from cookie", "error", err)

			c.JSON(http.StatusUnauthorized, "Unauthorized")
			return
		}

		DecodedRefreshToken, err := base64.StdEncoding.DecodeString(EncodedRefreshToken)
		if err != nil {
			logHandler.Error("failed to get decode refresh token", "error", err)

			c.JSON(http.StatusUnauthorized, "Unauthorized")
			return
		}

		blocked, err = storager.IsBlocked(ctx, string(DecodedRefreshToken))
		if err != nil {
			logHandler.Error("failed to check blocked token", "error", err)

			c.JSON(http.StatusUnauthorized, "Unauthorized")
			return
		}

		if blocked {
			logHandler.Error("refresh token in blacklist", "error", err)

			c.JSON(http.StatusUnauthorized, "Unauthorized")
			return
		}

		refreshToken, err := libJwt.VerifyToken(string(DecodedRefreshToken), "refresh")
		if err != nil {
			logHandler.Error("failed to verify refresh token", "error", err)
			c.JSON(http.StatusUnauthorized, "Unauthorized")
			return
		}

		RefreshClaims := refreshToken.Claims

		jwtClaims, ok = RefreshClaims.(jwt.MapClaims)
		if !ok {
			logHandler.Error("unexepted type of claims")

			c.JSON(http.StatusUnauthorized, "Unauthorized")
			return
		}

		guidRefresh := jwtClaims["guid"].(string)
		createdAtRefresh := jwtClaims["created_at"].(float64)

		// достать guid из обоих токенов сравнить их
		if guidAccess != guidRefresh {
			logHandler.Error("not pair token", "filter", "guid")

			c.JSON(http.StatusUnauthorized, "Unauthorized")
			return
		}
		// Проверить что они были созданы в одно время
		if !EqualWithinOneMinute(createdAtAccess, createdAtRefresh) {
			logHandler.Error("not pair token", "filter", "created_at")

			c.JSON(http.StatusUnauthorized, "Unauthorized")
			return
		}

		UserInfo, id, err := storager.FindByGUID(ctx, guidAccess)
		if err != nil {
			logHandler.Error("failed to get guid", "error", err.Error())

			c.JSON(http.StatusUnauthorized, "Unauthorized")
			return
		}

		err = bcrypt.CompareHashAndPassword([]byte(UserInfo.UserAgentHash), []byte(c.Request.UserAgent()))
		if err != nil {
			log.Error("Different User Agnet")

			log.Info("LOGOUTED")
			c.JSON(http.StatusUnauthorized, "Unauthorized")
			return
		}

		err = bcrypt.CompareHashAndPassword([]byte(UserInfo.IPhash), []byte(c.ClientIP()))
		if err != nil {
			log.Warn("Different IP")

			go webHook(logHandler, c.ClientIP())

		}

		idString := strconv.Itoa(id)

		err = storager.BlockToken(ctx, refreshToken.Raw, idString)
		if err != nil {
			log.Error("failed to block refresh token", "error", err)

			c.JSON(http.StatusInternalServerError, response.Error("Internal Error"))
		}

		err = storager.BlockToken(ctx, accessToken.Raw, idString)
		if err != nil {
			log.Error("failed to block access token", "error", err)

			c.JSON(http.StatusInternalServerError, response.Error("Internal Error"))
		}

		// ВЫДАЧА НОВЫХ ТОКЕНОВ
		accToken, err := libJwt.NewAccessToken(guidAccess, liveAccess)
		if err != nil {
			logHandler.Error("failed to generate jwt", "error", err.Error())

			logHandler.Debug("debug", "guid", guidAccess)

			c.JSON(http.StatusInternalServerError, response.Error("Internal error"))
			return
		}

		refToken, err := libJwt.NewRefreshToken(guidAccess, liveRefresh)
		if err != nil {
			logHandler.Error("failed to generate jwt", "error", err.Error())

			logHandler.Debug("debug", "guid", guidAccess)

			c.JSON(http.StatusInternalServerError, response.Error("Internal error"))
			return
		}

		hashedRefToken, err := libJwt.HashJWTbcrypt(refToken) // рефреш токен сначала сжимается до 64 байт чтобы затем захешировать его в bcrypt (ограничение 72 байта)
		if err != nil {
			logHandler.Error("failed to hash refresh token", "error", err.Error())

			logHandler.Debug("debug", "guid", guidAccess)

			c.JSON(http.StatusInternalServerError, response.Error("Internal error"))
			return
		}

		hashedUserAgent, err := bcrypt.GenerateFromPassword([]byte(c.Request.UserAgent()), bcrypt.DefaultCost)
		if err != nil {
			logHandler.Error("failed to hash user agent", "error", err.Error())

			logHandler.Debug("debug", "guid", guidAccess)

			c.JSON(http.StatusInternalServerError, response.Error("Internal error"))
			return
		}

		hashedIP, err := bcrypt.GenerateFromPassword([]byte(c.ClientIP()), bcrypt.DefaultCost)
		if err != nil {
			logHandler.Error("failed to hash user agent", "error", err.Error())

			logHandler.Debug("debug", "guid", guidAccess)

			c.JSON(http.StatusInternalServerError, response.Error("Internal error"))
			return
		}

		UserInfo = &models.UserInfo{
			GUID:          guidAccess,
			UserAgentHash: string(hashedUserAgent),
			IPhash:        string(hashedIP),
			TokenHash:     hashedRefToken,
		}

		if err := storager.UpdateUserInfo(ctx, UserInfo); err != nil {
			if errors.Is(err, storage.ErrGuidExists) {
				logHandler.Error(err.Error())

				logHandler.Debug("debug", "guid", guidAccess)

				c.JSON(http.StatusBadRequest, response.Error("GUID is already exists"))

				return

			}
			logHandler.Error("failed to save hash User Info", "error", err.Error())

			logHandler.Debug("debug", "guid", guidAccess)

			c.JSON(http.StatusInternalServerError, response.Error("Internal error"))
			return
		}

		c.SetCookie(
			"refreshToken", base64.URLEncoding.EncodeToString([]byte(refToken)),
			int(liveRefresh.Seconds()),
			"/",
			"localhost",
			false,
			true,
		)

		c.JSON(http.StatusOK, Response{Resp: response.OK(), AccessToken: accToken})

		// Проверить Юзер Агент. Если неверно = дееавторизовать

		// Если новый айпи = отправить POST на вебхук

		// Выдать новую пару токенов

		// Добавить в блек лист акцесс и рефреш токен

		// Обновить рефреш токен в основной таблице
	}
}

func EqualWithinOneMinute(t1, t2 float64) bool {
	diff := math.Abs(t1 - t2)
	return diff <= 60.0 // 1 минута = 60 секунд
}
