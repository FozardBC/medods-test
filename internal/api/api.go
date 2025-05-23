package api

import (
	"log/slog"
	_ "medods-test/docs"
	"medods-test/internal/api/handlers/auth/logout"
	"medods-test/internal/api/handlers/auth/token/refresh"
	"medods-test/internal/api/handlers/auth/token/tokens"
	"medods-test/internal/api/handlers/me"
	"medods-test/internal/api/middlewares/auth"
	"medods-test/internal/storage"

	"github.com/gin-contrib/requestid"
	"github.com/gin-gonic/gin"
	httpSwagger "github.com/swaggo/http-swagger"
)

type API struct {
	Router  *gin.Engine
	Storage storage.Storage
	Log     *slog.Logger
}

func New(log *slog.Logger, storage storage.Storage) *API {
	api := &API{
		Router:  gin.New(),
		Storage: storage,
		Log:     log,
	}

	api.Endpoints()

	return api
}

func (api *API) Endpoints() {

	v1 := api.Router.Group("api/v1/")

	v1.Use(requestid.New())
	v1.Use(gin.Logger())

	authV1 := v1.Group("/auth")
	authV1.POST("/token", tokens.New(api.Log, api.Storage))
	authV1.POST("/refresh", refresh.New(api.Log, api.Storage))
	authV1.PUT("/logout", logout.New(api.Log, api.Storage))

	v1.GET("/me", auth.AuthMiddleware(api.Log, api.Storage), me.New(api.Log))

	v1.GET("/swagger/*any", gin.WrapH(httpSwagger.Handler()))

}
