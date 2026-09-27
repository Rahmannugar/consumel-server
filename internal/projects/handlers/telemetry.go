package handlers

import (
	"github.com/Rahmannugar/consumel-server/internal/infra/telemetry"
	"github.com/gin-gonic/gin"
)

func RequestTelemetry() gin.HandlerFunc {
	return func(context *gin.Context) {
		telemetry.SetRequestOperation(
			context.Request.Context(),
			"projects.list",
			telemetry.CompletionDetails{
				Success: telemetry.Completion{
					Event: "projects.listed", Message: "Projects loaded",
				},
				Rejected: telemetry.Completion{
					Event: "projects.list.rejected", Message: "Project list request rejected",
				},
				Failed: telemetry.Completion{
					Event: "projects.list.failed", Message: "Could not load projects",
				},
			},
		)
		context.Next()
	}
}
