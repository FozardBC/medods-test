package auth

import (
	"context"
	"log/slog"
	"medods-test/internal/lib/api/response"
	jwtLib "medods-test/internal/lib/jwt"
	"net/http"
	"strings"

	"github.com/gin-contrib/requestid"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt"
)

type Provider interface {
	IsActive(ctx context.Context, guid string) (bool, error)
}

func AuthMiddleware(log *slog.Logger, provider Provider) gin.HandlerFunc {
	return func(c *gin.Context) {

		ctx := c.Request.Context()

		logHandler := log.With(
			"requestID", requestid.Get(c),
		)

		authHeader := c.Request.Header.Get("Authorization")
		if authHeader == "" {
			logHandler.Error("failed to get token from headers")
			c.AbortWithStatus(http.StatusUnauthorized)

			return
		}

		authTokens := strings.Split(authHeader, " ")

		if len(authTokens) != 2 || authTokens[0] != "Bearer" {
			logHandler.Error("failed to get beraer")
			c.AbortWithStatus(http.StatusUnauthorized)
		}

		token, err := jwtLib.VerifyToken(authTokens[1])
		if err != nil {

			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}

		var guidString string

		claimGUID := token.Claims.(jwt.MapClaims)
		guid := claimGUID["guid"]

		switch g := guid.(type) {
		case string:
			guidString = string(g)
		default:
			logHandler.Error("unexpected type of GUID im claims %T", g)

			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}

		active, err := provider.IsActive(ctx, guidString)
		if err != nil {
			logHandler.Error("failed to check active status", "error", err.Error())

			c.JSON(http.StatusInternalServerError, response.Error("Internal Error"))
		}
		if !active {
			logHandler.Info("Unauthorized")

			c.AbortWithStatus(http.StatusUnauthorized)
		}

		c.Set("guid", guid)

		c.Next()
	}

}
