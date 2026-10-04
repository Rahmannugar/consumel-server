package handlers

import (
	"github.com/Rahmannugar/consumel-server/internal/infra/telemetry"
	"github.com/gin-gonic/gin"
)

func balanceTelemetry(operation, successEvent, successMessage string) gin.HandlerFunc {
	return func(context *gin.Context) {
		telemetry.SetRequestOperation(context.Request.Context(), operation, telemetry.CompletionDetails{
			Success:  telemetry.Completion{Event: successEvent, Message: successMessage},
			Rejected: telemetry.Completion{Event: operation + ".rejected", Message: "Balance request rejected"},
			Failed:   telemetry.Completion{Event: operation + ".failed", Message: "Balance request failed"},
		})
		context.Next()
	}
}

func consumeTelemetry() gin.HandlerFunc {
	return func(context *gin.Context) {
		telemetry.SetRequestOperation(context.Request.Context(), "consumption.consume", telemetry.CompletionDetails{
			Success:  telemetry.Completion{Event: "usage.consumed", Message: "Usage consumed"},
			Rejected: telemetry.Completion{Event: "consumption.consume.rejected", Message: "Consume request rejected"},
			Failed:   telemetry.Completion{Event: "consumption.consume.failed", Message: "Consume request failed"},
		})
		context.Next()
	}
}

func operationTelemetry() gin.HandlerFunc {
	return func(context *gin.Context) {
		telemetry.SetRequestOperation(context.Request.Context(), "operations.list", telemetry.CompletionDetails{
			Success:  telemetry.Completion{Event: "operations.listed", Message: "Events loaded"},
			Rejected: telemetry.Completion{Event: "operations.list.rejected", Message: "Event list request rejected"},
			Failed:   telemetry.Completion{Event: "operations.list.failed", Message: "Event list request failed"},
		})
		context.Next()
	}
}

func analyticsTelemetry(operation string) gin.HandlerFunc {
	return func(context *gin.Context) {
		telemetry.SetRequestOperation(context.Request.Context(), operation, telemetry.CompletionDetails{
			Success:  telemetry.Completion{Event: "analytics.loaded", Message: "Analytics loaded"},
			Rejected: telemetry.Completion{Event: "analytics.get.rejected", Message: "Analytics request rejected"},
			Failed:   telemetry.Completion{Event: "analytics.get.failed", Message: "Analytics request failed"},
		})
		context.Next()
	}
}

func balancePathParameters() gin.HandlerFunc {
	return func(context *gin.Context) {
		context.Request.SetPathValue("customerID", context.Param("customerID"))
		context.Request.SetPathValue("meterKey", context.Param("meterKey"))
		context.Next()
	}
}

func balanceDashboardPathParameters() gin.HandlerFunc {
	return func(context *gin.Context) {
		context.Request.SetPathValue("projectID", context.Param("projectID"))
		context.Request.SetPathValue("environment", context.Param("environment"))
		context.Request.SetPathValue("customerID", context.Param("customerID"))
		context.Request.SetPathValue("meterKey", context.Param("meterKey"))
		context.Next()
	}
}
