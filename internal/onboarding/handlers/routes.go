package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"

	authenticationhandlers "github.com/Rahmannugar/consumel-server/internal/authentication/handlers"
	authenticationservices "github.com/Rahmannugar/consumel-server/internal/authentication/services"
	"github.com/Rahmannugar/consumel-server/internal/common/httpresponse"
	"github.com/Rahmannugar/consumel-server/internal/infra/telemetry"
	onboardingmodels "github.com/Rahmannugar/consumel-server/internal/onboarding/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type OnboardingService interface {
	Setup(
		context.Context,
		string,
		uuid.UUID,
		onboardingmodels.SetupRequest,
	) (onboardingmodels.Setup, error)
}

type Handler struct {
	resolver authenticationhandlers.TenantResolver
	service  OnboardingService
	logger   *slog.Logger
}

func RegisterRoutes(
	router gin.IRouter,
	resolver authenticationhandlers.TenantResolver,
	service OnboardingService,
	logger *slog.Logger,
) {
	handler := &Handler{resolver: resolver, service: service, logger: logger}
	router.POST("/onboarding", RequestTelemetry(), gin.WrapF(handler.Setup))
}

func (handler *Handler) Setup(response http.ResponseWriter, request *http.Request) {
	tenant, err := handler.resolver.Resolve(request)
	if err != nil {
		if errors.Is(err, authenticationservices.ErrUnauthenticated) {
			writeError(response, http.StatusUnauthorized, "not_authenticated",
				"Sign in to create your first project.")
			return
		}
		handler.fail(response, request, err)
		return
	}

	var input onboardingmodels.SetupRequest
	decoder := json.NewDecoder(http.MaxBytesReader(response, request.Body, 16<<10))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&input); err != nil {
		writeError(response, http.StatusBadRequest, "invalid_request",
			"Enter a valid organization name and project name.")
		return
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		writeError(response, http.StatusBadRequest, "invalid_request",
			"Send one JSON object.")
		return
	}

	setup, err := handler.service.Setup(
		request.Context(), tenant.Session.SubjectID, tenant.User.ID, input,
	)
	if err != nil {
		switch {
		case errors.Is(err, onboardingmodels.ErrOrganizationNameRequired),
			errors.Is(err, onboardingmodels.ErrOrganizationNameTooLong),
			errors.Is(err, onboardingmodels.ErrProjectNameRequired),
			errors.Is(err, onboardingmodels.ErrProjectNameTooLong):
			writeError(response, http.StatusBadRequest, "invalid_request",
				"Enter an organization name and project name between 1 and 120 characters.")
		case errors.Is(err, onboardingmodels.ErrExistingOrganizationWithoutProject):
			writeError(response, http.StatusConflict, "organization_already_exists",
				"This account already belongs to an organization that needs attention.")
		default:
			handler.fail(response, request, err)
		}
		return
	}

	telemetry.AddRequestLogAttributes(request.Context(),
		slog.String("organization_id", setup.Organization.ID.String()),
		slog.String("project_id", setup.Project.ID.String()),
		slog.Bool("created", setup.Created),
	)
	status := http.StatusOK
	if setup.Created {
		status = http.StatusCreated
	}
	if err := httpresponse.WriteJSON(response, status, setupResponse(setup)); err != nil {
		handler.logger.ErrorContext(request.Context(), "Could not send onboarding response",
			"event", "onboarding.response.failed",
			"operation", "onboarding.setup.respond",
			"outcome", "error",
			"error", err,
		)
	}
}

func (handler *Handler) fail(
	response http.ResponseWriter,
	request *http.Request,
	err error,
) {
	handler.logger.ErrorContext(request.Context(), "Could not complete onboarding",
		"event", "onboarding.setup.failed",
		"operation", "onboarding.setup",
		"outcome", "error",
		"error", err,
	)
	writeError(response, http.StatusInternalServerError, "onboarding_failed",
		"Consumel could not create your project. Try again shortly.")
}

func writeError(response http.ResponseWriter, status int, code, message string) {
	_ = httpresponse.WriteError(response, status, code, message)
}
