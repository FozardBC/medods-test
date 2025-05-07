package logout

import (
	"context"
	"log/slog"
	"medods-test/internal/lib/api/response"
	libJwt "medods-test/internal/lib/jwt"
	"medods-test/internal/models"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-contrib/requestid"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt"
)

type Storage interface {
	BlockToken(ctx context.Context, hashedToken string, idToken string) error
	IsBlocked(ctx context.Context, hashedToken string) (bool, error)
	FindByGUID(ctx context.Context, guid string) (*models.UserInfo, int, error)
	Logout(ctx context.Context, guid string) error
}

func New(log *slog.Logger, storage Storage) gin.HandlerFunc {
	return func(c *gin.Context) {

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

		token, err := libJwt.VerifyToken(authTokens[1], "access")
		if err != nil {
			logHandler.Error("failed to verify token")
			c.JSON(http.StatusUnauthorized, "Unauthorized")
			return
		}

		claims := token.Claims

		jwtClaims, ok := claims.(jwt.MapClaims)
		if !ok {
			logHandler.Error("unexepted type of claims")

			c.JSON(http.StatusUnauthorized, "Unauthorized")
			return
		}

		guid := jwtClaims["guid"].(string)

		UserInfo, id, err := storage.FindByGUID(ctx, guid)
		if err != nil {
			logHandler.Error("failed to find UserInfo", "error", err)

			c.JSON(http.StatusUnauthorized, "Unauthorized")
			return
		}

		IDstring := strconv.Itoa(id)

		err = storage.BlockToken(ctx, UserInfo.TokenHash, IDstring)
		if err != nil {
			logHandler.Error("failed to block acc token", "error", err)

			c.JSON(http.StatusInternalServerError, "Internal error")
			return
		}

		hashedAccessToken, err := libJwt.HashJWTbcrypt(token.Raw)
		if err != nil {
			logHandler.Error("failed to hash access token", "error", err)

			c.JSON(http.StatusInternalServerError, "Internal error")
		}

		err = storage.BlockToken(ctx, hashedAccessToken, IDstring)
		if err != nil {
			logHandler.Error("failed to block acc token", "error", err)

			c.JSON(http.StatusInternalServerError, "Internal error")
			return
		}

		err = storage.Logout(ctx, guid)
		if err != nil {
			logHandler.Error("failed to logout", "error", err)

			c.JSON(http.StatusInternalServerError, "Internal error")

			return
		}

		c.JSON(http.StatusOK, response.OK())

	}
}
