package health

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type response struct {
	Status string `json:"status"`
}

func RegisterRoutes(router gin.IRoutes) {
	router.GET("/health/live", live)
	router.GET("/health/ready", ready)
}

func live(ctx *gin.Context) {
	ctx.JSON(http.StatusOK, response{Status: "ok"})
}

func ready(ctx *gin.Context) {
	ctx.JSON(http.StatusOK, response{Status: "ok"})
}
