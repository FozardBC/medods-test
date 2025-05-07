package me

import (
	"log/slog"
	"medods-test/internal/lib/api/response"
	"net/http"

	"github.com/gin-contrib/requestid"
	"github.com/gin-gonic/gin"
)

type Response struct {
	GUID string `json:"guid"`
}

func New(log *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {

		logHandler := log.With(
			"requestID", requestid.Get(c),
		)

		var Response Response

		guid, ok := c.Get("guid")
		if !ok {
			logHandler.Error("failed to get guid from context")

			c.JSON(http.StatusInternalServerError, "Internal error")
			return
		}

		switch g := guid.(type) {
		case string:
			Response.GUID = g
		default:
			logHandler.Error("unexpected type of GUID %T", g)

			c.JSON(http.StatusBadRequest, response.Error("unexpected type of GUID"))
		}

		c.JSON(http.StatusOK, Response)

	}
}
