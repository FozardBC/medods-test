package me

import (
	"log/slog"
	"net/http"

	"github.com/gin-contrib/requestid"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt"
)

type Response struct {
	GUID string `json:"guid"`
}

type Claims struct {
	GUID string `json:"guid"`
}

func New(log *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {

		logHandler := log.With(
			"requestID", requestid.Get(c),
		)

		var Response Response

		claims, ok := c.Get("claims")
		if !ok {
			logHandler.Error("failed to get guid from context")

			c.JSON(http.StatusUnauthorized, "Unauthorized")
			return
		}

		jwtClaims, ok := claims.(jwt.MapClaims)
		if !ok {
			logHandler.Error("unexepted type of claims")

			c.JSON(http.StatusUnauthorized, "Unauthorized")
			return
		}

		Response.GUID = jwtClaims["guid"].(string)
		c.JSON(http.StatusOK, Response)

	}
}
