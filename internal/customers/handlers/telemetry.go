package handlers

import (
	"github.com/Rahmannugar/consumel-server/internal/infra/telemetry"
	"github.com/gin-gonic/gin"
)

func customerTelemetry(operation, successEvent, successMessage string) gin.HandlerFunc {
	return func(context *gin.Context) {
		telemetry.SetRequestOperation(context.Request.Context(), operation, telemetry.CompletionDetails{
			Success:  telemetry.Completion{Event: successEvent, Message: successMessage},
			Rejected: telemetry.Completion{Event: operation + ".rejected", Message: successMessage + " request rejected"},
			Failed:   telemetry.Completion{Event: operation + ".failed", Message: successMessage + " failed"},
		})
		context.Next()
	}
}
