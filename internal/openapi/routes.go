package openapi

import (
	_ "embed"
	"net/http"

	"github.com/Rahmannugar/consumel-server/internal/openapi/clients"
	"github.com/gin-gonic/gin"
)

//go:embed assets/favicon.ico
var favicon []byte

func RegisterRoutes(router gin.IRoutes) error {
	document, err := Document()
	if err != nil {
		return err
	}
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
