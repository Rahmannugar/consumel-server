package openapi

import (
	_ "embed"
	"net/http"

	"github.com/Rahmannugar/consumel-server/internal/openapi/clients"
	"github.com/gin-gonic/gin"
)

//go:embed assets/favicon.ico
var favicon []byte

//go:embed swagger.json
var document []byte

func RegisterRoutes(router gin.IRoutes) error {
	router.GET("/favicon.ico", func(context *gin.Context) {
		context.Data(http.StatusOK, "image/x-icon", favicon)
	})
	router.GET("/openapi.json", func(context *gin.Context) {
		context.Data(http.StatusOK, "application/json; charset=utf-8", document)
	})
	router.GET("/docs", func(context *gin.Context) {
		context.Data(http.StatusOK, "text/html; charset=utf-8", []byte(clients.ScalarHTML("/openapi.json")))
	})
	return nil
}
