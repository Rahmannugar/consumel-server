package handlers

import (
	"github.com/Rahmannugar/consumel-server/internal/infra/telemetry"
	"github.com/gin-gonic/gin"
)

func RequestTelemetry(operation string, details telemetry.CompletionDetails) gin.HandlerFunc {
	return func(context *gin.Context) {
		telemetry.SetRequestOperation(context.Request.Context(), operation, details)
		context.Next()
	}
}

func projectListTelemetry() gin.HandlerFunc {
	return RequestTelemetry("projects.list", telemetry.CompletionDetails{
		Success:  telemetry.Completion{Event: "projects.listed", Message: "Projects loaded"},
		Rejected: telemetry.Completion{Event: "projects.list.rejected", Message: "Project list request rejected"},
		Failed:   telemetry.Completion{Event: "projects.list.failed", Message: "Could not load projects"},
	})
}

func projectCreateTelemetry() gin.HandlerFunc {
	return RequestTelemetry("projects.create", telemetry.CompletionDetails{
		Success:  telemetry.Completion{Event: "projects.created", Message: "Project created"},
		Rejected: telemetry.Completion{Event: "projects.create.rejected", Message: "Project creation rejected"},
		Failed:   telemetry.Completion{Event: "projects.create.failed", Message: "Could not create project"},
	})
}

func apiKeyTelemetry(operation, event, message string) gin.HandlerFunc {
	return RequestTelemetry(operation, telemetry.CompletionDetails{
		Success:  telemetry.Completion{Event: event, Message: message},
		Rejected: telemetry.Completion{Event: event + ".rejected", Message: message + " request rejected"},
		Failed:   telemetry.Completion{Event: event + ".failed", Message: message + " failed"},
	})
}
