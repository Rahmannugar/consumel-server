package handlers

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	authenticationhandlers "github.com/Rahmannugar/consumel-server/internal/authentication/handlers"
	authenticationservices "github.com/Rahmannugar/consumel-server/internal/authentication/services"
	"github.com/Rahmannugar/consumel-server/internal/common/httpresponse"
	"github.com/Rahmannugar/consumel-server/internal/infra/telemetry"
	projectmodels "github.com/Rahmannugar/consumel-server/internal/projects/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type ProjectService interface {
	ListProjects(context.Context, uuid.UUID) ([]projectmodels.Project, error)
}

type Handler struct {
	resolver authenticationhandlers.TenantResolver
	service  ProjectService
	logger   *slog.Logger
}

func RegisterRoutes(
	router gin.IRouter,
	resolver authenticationhandlers.TenantResolver,
	service ProjectService,
	logger *slog.Logger,
) {
	handler := &Handler{resolver: resolver, service: service, logger: logger}
	router.GET("/v1/projects", RequestTelemetry(), gin.WrapF(handler.List))
}

func (handler *Handler) List(response http.ResponseWriter, request *http.Request) {
	tenant, err := handler.resolver.Resolve(request)
	if err != nil {
		if errors.Is(err, authenticationservices.ErrUnauthenticated) {
			_ = httpresponse.WriteError(response, http.StatusUnauthorized, "not_authenticated",
				"Sign in to view your projects.")
			return
		}
		handler.fail(response, request, err)
		return
	}
	projects, err := handler.service.ListProjects(request.Context(), tenant.User.ID)
	if err != nil {
		handler.fail(response, request, err)
		return
	}
	telemetry.AddRequestLogAttributes(request.Context(),
		slog.String("user_id", tenant.User.ID.String()),
		slog.Int("project_count", len(projects)),
	)
	if err := httpresponse.WriteJSON(response, http.StatusOK, listResponse(projects)); err != nil {
		handler.logger.ErrorContext(request.Context(), "Could not send projects response",
			"event", "projects.response.failed",
			"operation", "projects.list.respond",
			"outcome", "error",
			"error", err,
		)
	}
}

func (handler *Handler) fail(response http.ResponseWriter, request *http.Request, err error) {
	handler.logger.ErrorContext(request.Context(), "Could not list projects",
		"event", "projects.list.failed",
		"operation", "projects.list",
		"outcome", "error",
		"error", err,
	)
	_ = httpresponse.WriteError(response, http.StatusInternalServerError, "projects_load_failed",
		"Consumel could not load your projects. Try again shortly.")
}
