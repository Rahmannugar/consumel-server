package handlers

import (
	"github.com/Rahmannugar/consumel-server/internal/infra/telemetry"
	"github.com/gin-gonic/gin"
)

func RequestTelemetry() gin.HandlerFunc {
	return func(context *gin.Context) {
		telemetry.SetRequestOperation(
			context.Request.Context(),
			"onboarding.setup",
			telemetry.CompletionDetails{
				Success: telemetry.Completion{
					Event: "onboarding.completed", Message: "First project created",
				},
				Rejected: telemetry.Completion{
					Event: "onboarding.rejected", Message: "Onboarding request rejected",
				},
				Failed: telemetry.Completion{
					Event: "onboarding.failed", Message: "Could not complete onboarding",
				},
			},
		)
		context.Next()
	}
}
