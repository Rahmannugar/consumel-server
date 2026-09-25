package repositories

import (
	"context"
	"fmt"

	authenticationdb "github.com/Rahmannugar/consumel-server/internal/authentication/repositories/generated"
	organizationmodels "github.com/Rahmannugar/consumel-server/internal/organizations/models"
	usermodels "github.com/Rahmannugar/consumel-server/internal/users/models"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type AccountContextRepository struct {
	queries *authenticationdb.Queries
}

func NewAccountContextRepository(pool *pgxpool.Pool) *AccountContextRepository {
	return &AccountContextRepository{queries: authenticationdb.New(pool)}
}

func (repository *AccountContextRepository) AccountContextByAuthlierSubjectID(
	ctx context.Context,
	authlierSubjectID string,
) (usermodels.User, []organizationmodels.OrganizationAccess, bool, error) {
	rows, err := repository.queries.GetAccountContextByAuthlierSubjectID(ctx, authlierSubjectID)
	if err != nil {
		return usermodels.User{}, nil, false, fmt.Errorf("get account context by Authlier subject ID: %w", err)
	}
	if len(rows) == 0 {
		return usermodels.User{}, nil, false, nil
	}

	user := usermodels.User{
		ID:                rows[0].UserID,
		AuthlierSubjectID: rows[0].AuthlierSubjectID,
		CreatedAt:         rows[0].UserCreatedAt.Time,
	}
	access := make([]organizationmodels.OrganizationAccess, 0, len(rows))
	for _, row := range rows {
		if !row.OrganizationID.Valid || row.OrganizationName == nil ||
			!row.RoleID.Valid || row.RoleName == nil {
			continue
		}

		var systemKey *organizationmodels.OrganizationRoleSystemKey
		if row.RoleSystemKey != nil {
			value := organizationmodels.OrganizationRoleSystemKey(*row.RoleSystemKey)
			systemKey = &value
		}
		access = append(access, organizationmodels.OrganizationAccess{
			OrganizationID:   uuid.UUID(row.OrganizationID.Bytes),
			OrganizationName: *row.OrganizationName,
			Owner:            row.Owner,
			RoleID:           uuid.UUID(row.RoleID.Bytes),
			RoleName:         *row.RoleName,
			RoleSystemKey:    systemKey,
		})
	}

	return user, access, true, nil
}
