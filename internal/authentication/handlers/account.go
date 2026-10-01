package handlers

import (
	"errors"
	"log/slog"
	"net/http"

	authenticationmodels "github.com/Rahmannugar/consumel-server/internal/authentication/models"
	authenticationservices "github.com/Rahmannugar/consumel-server/internal/authentication/services"
	"github.com/Rahmannugar/consumel-server/internal/common/httpresponse"
	"github.com/Rahmannugar/consumel-server/internal/infra/telemetry"
	"github.com/Rahmannugar/consumel-server/internal/openapi"
)

type TenantResolver interface {
	Resolve(*http.Request) (authenticationmodels.AuthenticatedTenant, error)
}

type AccountHandler struct {
	resolver TenantResolver
	logger   *slog.Logger
}

func NewAccountHandler(resolver TenantResolver, logger *slog.Logger) *AccountHandler {
	return &AccountHandler{resolver: resolver, logger: logger}
}

func (handler *AccountHandler) Get(response http.ResponseWriter, request *http.Request) {
	tenant, err := handler.resolver.Resolve(request)
	if err != nil {
		if errors.Is(err, authenticationservices.ErrUnauthenticated) {
			_ = httpresponse.WriteError(
				response,
				http.StatusUnauthorized,
				"unauthenticated",
				"Sign in to access your Consumel account.",
			)
			return
		}
		handler.logger.ErrorContext(request.Context(), "Could not load user account",
			"event", "user.account.load.failed",
			"operation", "user.account.load",
			"outcome", "error",
			"error", err,
		)
		_ = httpresponse.WriteError(
			response,
			http.StatusInternalServerError,
			"account_load_failed",
			"Consumel could not load your account. Try again shortly.",
		)
		return
	}

	organizations := make([]openapi.OrganizationAccess, 0, len(tenant.OrganizationAccess))
	telemetry.AddRequestLogAttributes(request.Context(),
		slog.String("user_id", tenant.User.ID.String()),
		slog.Int("organization_count", len(tenant.OrganizationAccess)),
	)
	for _, access := range tenant.OrganizationAccess {
		var systemKey *string
		if access.RoleSystemKey != nil {
			value := string(*access.RoleSystemKey)
			systemKey = &value
		}
		organizations = append(organizations, openapi.OrganizationAccess{
			ID:            access.OrganizationID.String(),
			Name:          access.OrganizationName,
			Owner:         access.Owner,
			RoleID:        access.RoleID.String(),
			RoleName:      access.RoleName,
			RoleSystemKey: systemKey,
		})
	}

	if err := httpresponse.WriteJSON(response, http.StatusOK, openapi.Account{
		Session: openapi.AccountSession{
			ID:        tenant.Session.ID,
			CreatedAt: tenant.Session.CreatedAt,
			ExpiresAt: tenant.Session.ExpiresAt,
		},
		User:          openapi.AccountUser{ID: tenant.User.ID.String(), Email: tenant.User.Email},
		Organizations: organizations,
	}); err != nil {
		handler.logger.ErrorContext(request.Context(), "Could not send user account response",
			"event", "user.account.response.failed",
			"operation", "user.account.respond",
			"outcome", "error",
			"error", err,
		)
	}
}
