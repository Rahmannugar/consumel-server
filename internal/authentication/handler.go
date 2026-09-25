package authentication

import (
	"errors"
	"log/slog"
	"net/http"
	"time"

	authenticationmodels "github.com/Rahmannugar/consumel-server/internal/authentication/models"
	authenticationservices "github.com/Rahmannugar/consumel-server/internal/authentication/services"
	"github.com/Rahmannugar/consumel-server/internal/common/httpresponse"
)

type TenantResolver interface {
	Resolve(*http.Request) (authenticationmodels.AuthenticatedTenant, error)
}

type Handler struct {
	resolver TenantResolver
	logger   *slog.Logger
}

type contextResponse struct {
	Session       sessionResponse              `json:"session"`
	User          userResponse                 `json:"user"`
	Organizations []organizationAccessResponse `json:"organizations"`
}

type sessionResponse struct {
	ID        string    `json:"id"`
	CreatedAt time.Time `json:"createdAt"`
	ExpiresAt time.Time `json:"expiresAt"`
}

type userResponse struct {
	ID string `json:"id"`
}

type organizationAccessResponse struct {
	ID            string  `json:"id"`
	Name          string  `json:"name"`
	Owner         bool    `json:"owner"`
	RoleID        string  `json:"roleId"`
	RoleName      string  `json:"roleName"`
	RoleSystemKey *string `json:"roleSystemKey"`
}

func NewHandler(resolver TenantResolver, logger *slog.Logger) *Handler {
	return &Handler{resolver: resolver, logger: logger}
}

func (handler *Handler) Context(response http.ResponseWriter, request *http.Request) {
	tenant, err := handler.resolver.Resolve(request)
	if err != nil {
		if errors.Is(err, authenticationservices.ErrUnauthenticated) {
			_ = httpresponse.WriteError(
				response,
				http.StatusUnauthorized,
				"unauthenticated",
				"Sign in to access your Consumel account context.",
			)
			return
		}
		handler.logger.ErrorContext(request.Context(), "resolve tenant authentication context",
			"event", "authentication.context.resolve_failed",
			"outcome", "error",
			"error", err,
		)
		_ = httpresponse.WriteError(
			response,
			http.StatusInternalServerError,
			"authentication_context_failed",
			"Consumel could not load your account context. Try again shortly.",
		)
		return
	}

	organizations := make([]organizationAccessResponse, 0, len(tenant.OrganizationAccess))
	for _, access := range tenant.OrganizationAccess {
		var systemKey *string
		if access.RoleSystemKey != nil {
			value := string(*access.RoleSystemKey)
			systemKey = &value
		}
		organizations = append(organizations, organizationAccessResponse{
			ID:            access.OrganizationID.String(),
			Name:          access.OrganizationName,
			Owner:         access.Owner,
			RoleID:        access.RoleID.String(),
			RoleName:      access.RoleName,
			RoleSystemKey: systemKey,
		})
	}

	if err := httpresponse.WriteJSON(response, http.StatusOK, contextResponse{
		Session: sessionResponse{
			ID:        tenant.Session.ID,
			CreatedAt: tenant.Session.CreatedAt,
			ExpiresAt: tenant.Session.ExpiresAt,
		},
		User:          userResponse{ID: tenant.User.ID.String()},
		Organizations: organizations,
	}); err != nil {
		handler.logger.ErrorContext(request.Context(), "write tenant authentication context",
			"event", "authentication.context.response_failed",
			"outcome", "error",
			"error", err,
		)
	}
}
