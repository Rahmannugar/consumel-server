package repositories

import (
	"context"
	"fmt"
	"time"

	"github.com/Rahmannugar/consumel-server/internal/infra/emaildelivery"
	onboardingmodels "github.com/Rahmannugar/consumel-server/internal/onboarding/models"
	organizationmodels "github.com/Rahmannugar/consumel-server/internal/organizations/models"
	organizationdb "github.com/Rahmannugar/consumel-server/internal/organizations/repositories/generated"
	projectmodels "github.com/Rahmannugar/consumel-server/internal/projects/models"
	projectdb "github.com/Rahmannugar/consumel-server/internal/projects/repositories/generated"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

const welcomeDeliveryLifetime = 24 * time.Hour

type EmailQueue interface {
	EnqueueTx(context.Context, pgx.Tx, string, emaildelivery.Payload) error
}

type Repository struct {
	pool  *pgxpool.Pool
	email EmailQueue
}

func New(pool *pgxpool.Pool, email EmailQueue) *Repository {
	return &Repository{pool: pool, email: email}
}

func (repository *Repository) SetupFirstProject(
	ctx context.Context,
	subjectID string,
	organization organizationmodels.Organization,
	roles []organizationmodels.OrganizationRole,
	membership organizationmodels.OrganizationMembership,
	project projectmodels.Project,
	environments []projectmodels.ProjectEnvironment,
	projectURL string,
) (onboardingmodels.Setup, error) {
	tx, err := repository.pool.Begin(ctx)
	if err != nil {
		return onboardingmodels.Setup{}, fmt.Errorf("begin onboarding transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// Locking the local user serializes retries for one account while allowing
	// unrelated accounts to onboard concurrently.
	var storedSubjectID string
	if err := tx.QueryRow(ctx,
		"SELECT authlier_subject_id FROM users WHERE id = $1 FOR UPDATE",
		membership.UserID,
	).Scan(&storedSubjectID); err != nil {
		return onboardingmodels.Setup{}, fmt.Errorf("lock onboarding user: %w", err)
	}
	if storedSubjectID != subjectID {
		return onboardingmodels.Setup{}, fmt.Errorf("authenticated subject does not match user")
	}

	existing, found, err := existingSetup(ctx, tx, membership.UserID)
	if err != nil {
		return onboardingmodels.Setup{}, err
	}
	if found {
		if err := tx.Commit(ctx); err != nil {
			return onboardingmodels.Setup{}, fmt.Errorf("commit idempotent onboarding lookup: %w", err)
		}
		return existing, nil
	}

	var currentOrganizationCount int
	if err := tx.QueryRow(ctx, `SELECT count(*)
		FROM organization_memberships memberships
		WHERE memberships.user_id = $1
		  AND memberships.removed_at IS NULL`, membership.UserID,
	).Scan(&currentOrganizationCount); err != nil {
		return onboardingmodels.Setup{}, fmt.Errorf("check existing organization access: %w", err)
	}
	if currentOrganizationCount > 0 {
		return onboardingmodels.Setup{}, onboardingmodels.ErrExistingOrganizationWithoutProject
	}

	var recipient string
	if err := tx.QueryRow(ctx,
		"SELECT email FROM authlier_users WHERE id = $1",
		subjectID,
	).Scan(&recipient); err != nil {
		return onboardingmodels.Setup{}, fmt.Errorf("resolve onboarding email recipient: %w", err)
	}

	organizationQueries := organizationdb.New(tx)
	projectQueries := projectdb.New(tx)
	createdOrganization, err := organizationQueries.CreateOrganization(
		ctx,
		organizationdb.CreateOrganizationParams{
			ID: organization.ID, OwnerUserID: organization.OwnerUserID, Name: organization.Name,
		},
	)
	if err != nil {
		return onboardingmodels.Setup{}, fmt.Errorf("create onboarding organization: %w", err)
	}
	for _, role := range roles {
		if _, err := organizationQueries.CreateOrganizationRole(
			ctx,
			organizationdb.CreateOrganizationRoleParams{
				ID: role.ID, OrganizationID: role.OrganizationID, Name: role.Name,
				SystemKey: nullableRoleSystemKey(role.SystemKey),
			},
		); err != nil {
			return onboardingmodels.Setup{}, fmt.Errorf("create onboarding role: %w", err)
		}
	}
	if _, err := organizationQueries.CreateOrganizationMembership(
		ctx,
		organizationdb.CreateOrganizationMembershipParams{
			OrganizationID: membership.OrganizationID,
			UserID:         membership.UserID,
			RoleID:         membership.RoleID,
			Status:         string(membership.Status),
		},
	); err != nil {
		return onboardingmodels.Setup{}, fmt.Errorf("create onboarding membership: %w", err)
	}

	createdProject, err := projectQueries.CreateProject(
		ctx,
		projectdb.CreateProjectParams{
			ID: project.ID, OrganizationID: project.OrganizationID, Name: project.Name,
			Slug: project.Slug,
		},
	)
	if err != nil {
		return onboardingmodels.Setup{}, fmt.Errorf("create onboarding project: %w", err)
	}
	createdEnvironments := make([]projectmodels.ProjectEnvironment, 0, len(environments))
	for _, environment := range environments {
		created, err := projectQueries.CreateProjectEnvironment(
			ctx,
			projectdb.CreateProjectEnvironmentParams{
				ID: environment.ID, ProjectID: environment.ProjectID,
				Environment: string(environment.Name),
				ActivatedAt: nullableTimestamp(environment.ActivatedAt),
			},
		)
		if err != nil {
			return onboardingmodels.Setup{}, fmt.Errorf("create onboarding environment: %w", err)
		}
		createdEnvironments = append(createdEnvironments, mapEnvironment(created))
	}

	if err := repository.email.EnqueueTx(ctx, tx, emaildelivery.TemplateWelcome, emaildelivery.Payload{
		SubjectID: subjectID, Recipient: recipient, URL: projectURL,
		OrganizationName: organization.Name, ProjectName: project.Name,
		ExpiresAt: time.Now().UTC().Add(welcomeDeliveryLifetime),
	}); err != nil {
		return onboardingmodels.Setup{}, fmt.Errorf("queue welcome email: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return onboardingmodels.Setup{}, fmt.Errorf("commit onboarding: %w", err)
	}
	return onboardingmodels.Setup{
		Organization: mapOrganization(createdOrganization),
		Project: projectmodels.Project{
			ID: createdProject.ID, OrganizationID: createdProject.OrganizationID,
			Name: createdProject.Name, Slug: createdProject.Slug,
			CreatedAt: createdProject.CreatedAt.Time,
			UpdatedAt: createdProject.UpdatedAt.Time, Environments: createdEnvironments,
		},
		Created: true,
	}, nil
}

func existingSetup(
	ctx context.Context,
	tx pgx.Tx,
	userID uuid.UUID,
) (onboardingmodels.Setup, bool, error) {
	rows, err := tx.Query(ctx, `WITH first_project AS (
			SELECT projects.id
			FROM organization_memberships memberships
			JOIN organizations ON organizations.id = memberships.organization_id
			JOIN projects ON projects.organization_id = organizations.id
			WHERE memberships.user_id = $1
			  AND memberships.status = 'active'
			  AND memberships.removed_at IS NULL
			  AND organizations.deleted_at IS NULL
			  AND organizations.suspended_at IS NULL
			ORDER BY organizations.created_at, projects.created_at, projects.id
			LIMIT 1
		)
		SELECT
			organizations.id, organizations.owner_user_id, organizations.name,
			organizations.created_at, organizations.updated_at,
			projects.id, projects.name, projects.slug, projects.created_at, projects.updated_at,
			project_environments.id, project_environments.environment,
			project_environments.activated_at, project_environments.created_at
		FROM organization_memberships memberships
		JOIN organizations ON organizations.id = memberships.organization_id
		JOIN projects ON projects.organization_id = organizations.id
		JOIN project_environments ON project_environments.project_id = projects.id
		JOIN first_project ON first_project.id = projects.id
		WHERE memberships.user_id = $1
		  AND memberships.status = 'active'
		  AND memberships.removed_at IS NULL
		  AND organizations.deleted_at IS NULL
		  AND organizations.suspended_at IS NULL
		ORDER BY CASE project_environments.environment WHEN 'sandbox' THEN 0 ELSE 1 END`,
		userID,
	)
	if err != nil {
		return onboardingmodels.Setup{}, false, fmt.Errorf("find existing onboarding setup: %w", err)
	}
	defer rows.Close()

	var setup onboardingmodels.Setup
	for rows.Next() {
		var environment projectmodels.ProjectEnvironment
		var environmentName string
		var activatedAt pgtype.Timestamptz
		if err := rows.Scan(
			&setup.Organization.ID, &setup.Organization.OwnerUserID, &setup.Organization.Name,
			&setup.Organization.CreatedAt, &setup.Organization.UpdatedAt,
			&setup.Project.ID, &setup.Project.Name, &setup.Project.Slug,
			&setup.Project.CreatedAt, &setup.Project.UpdatedAt,
			&environment.ID, &environmentName, &activatedAt, &environment.CreatedAt,
		); err != nil {
			return onboardingmodels.Setup{}, false, fmt.Errorf("scan existing onboarding setup: %w", err)
		}
		setup.Project.OrganizationID = setup.Organization.ID
		environment.ProjectID = setup.Project.ID
		environment.Name = projectmodels.ProjectEnvironmentName(environmentName)
		environment.ActivatedAt = nullableTime(activatedAt)
		setup.Project.Environments = append(setup.Project.Environments, environment)
	}
	if err := rows.Err(); err != nil {
		return onboardingmodels.Setup{}, false, fmt.Errorf("read existing onboarding setup: %w", err)
	}
	return setup, setup.Project.ID != uuid.Nil, nil
}

func nullableRoleSystemKey(
	value *organizationmodels.OrganizationRoleSystemKey,
) *string {
	if value == nil {
		return nil
	}
	systemKey := string(*value)
	return &systemKey
}

func nullableTimestamp(value *time.Time) pgtype.Timestamptz {
	if value == nil {
		return pgtype.Timestamptz{}
	}
	return pgtype.Timestamptz{Time: *value, Valid: true}
}

func nullableTime(value pgtype.Timestamptz) *time.Time {
	if !value.Valid {
		return nil
	}
	timestamp := value.Time
	return &timestamp
}

func mapEnvironment(environment projectdb.ProjectEnvironment) projectmodels.ProjectEnvironment {
	return projectmodels.ProjectEnvironment{
		ID: environment.ID, ProjectID: environment.ProjectID,
		Name:        projectmodels.ProjectEnvironmentName(environment.Environment),
		ActivatedAt: nullableTime(environment.ActivatedAt), CreatedAt: environment.CreatedAt.Time,
	}
}

func mapOrganization(organization organizationdb.CreateOrganizationRow) organizationmodels.Organization {
	return organizationmodels.Organization{
		ID: organization.ID, OwnerUserID: organization.OwnerUserID, Name: organization.Name,
		CreatedAt: organization.CreatedAt.Time, UpdatedAt: organization.UpdatedAt.Time,
		DeletedAt:   nullableTime(organization.DeletedAt),
		SuspendedAt: nullableTime(organization.SuspendedAt),
	}
}
