package repositories

import (
	"context"
	"errors"
	"fmt"

	customermodels "github.com/Rahmannugar/consumel-server/internal/customers/models"
	customerdb "github.com/Rahmannugar/consumel-server/internal/customers/repositories/generated"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type CustomerRepository struct {
	queries *customerdb.Queries
}

func NewCustomerRepository(pool *pgxpool.Pool) *CustomerRepository {
	return &CustomerRepository{queries: customerdb.New(pool)}
}

func (repository *CustomerRepository) Create(
	ctx context.Context,
	customer customermodels.Customer,
) (customermodels.Customer, error) {
	created, err := repository.queries.CreateCustomer(ctx, customerdb.CreateCustomerParams{
		ID: customer.ID, ProjectEnvironmentID: customer.ProjectEnvironmentID,
		CustomerID: customer.CustomerID, Name: customer.Name, Email: customer.Email,
		MetadataPlan: customer.Metadata.Plan, MetadataCountry: customer.Metadata.Country,
		MetadataLocation: customer.Metadata.Location,
	})
	if customerExists(err) {
		return customermodels.Customer{}, customermodels.ErrCustomerExists
	}
	if err != nil {
		return customermodels.Customer{}, fmt.Errorf("create customer: %w", err)
	}
	return mapCustomer(created), nil
}

func (repository *CustomerRepository) Get(
	ctx context.Context,
	projectEnvironmentID uuid.UUID,
	customerID string,
) (customermodels.Customer, error) {
	row, err := repository.queries.CustomerByPublicID(ctx, customerdb.CustomerByPublicIDParams{
		ProjectEnvironmentID: projectEnvironmentID, CustomerID: customerID,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return customermodels.Customer{}, customermodels.ErrCustomerNotFound
	}
	if err != nil {
		return customermodels.Customer{}, fmt.Errorf("get customer: %w", err)
	}
	return mapCustomer(row), nil
}

func (repository *CustomerRepository) List(
	ctx context.Context,
	projectEnvironmentID uuid.UUID,
	cursor *customermodels.ListCursor,
	limit int,
	search string,
) ([]customermodels.Customer, *customermodels.ListCursor, error) {
	params := customerdb.ListCustomersParams{
		ProjectEnvironmentID: projectEnvironmentID, PageSize: int32(limit + 1),
	}
	if cursor != nil {
		params.CursorCreatedAt = pgtype.Timestamptz{Time: cursor.CreatedAt, Valid: true}
		params.CursorID = pgtype.UUID{Bytes: cursor.ID, Valid: true}
	}
	var rows []customerdb.Customer
	var err error
	if search == "" {
		rows, err = repository.queries.ListCustomers(ctx, params)
	} else {
		rows, err = repository.queries.SearchCustomers(ctx, customerdb.SearchCustomersParams{
			SearchProjectEnvironmentID: params.ProjectEnvironmentID,
			SearchQuery:                search, CursorCreatedAt: params.CursorCreatedAt,
			CursorID: params.CursorID, PageSize: params.PageSize,
		})
	}
	if err != nil {
		return nil, nil, fmt.Errorf("list customers: %w", err)
	}
	customers := make([]customermodels.Customer, 0, min(len(rows), limit))
	for index, row := range rows {
		if index == limit {
			last := customers[len(customers)-1]
			return customers, &customermodels.ListCursor{CreatedAt: last.CreatedAt, ID: last.ID}, nil
		}
		customers = append(customers, mapCustomer(row))
	}
	return customers, nil, nil
}

func (repository *CustomerRepository) Update(
	ctx context.Context,
	projectEnvironmentID uuid.UUID,
	customerID string,
	request customermodels.UpdateCustomerRequest,
) (customermodels.Customer, error) {
	row, err := repository.queries.UpdateCustomer(ctx, customerdb.UpdateCustomerParams{
		ProjectEnvironmentID: projectEnvironmentID, CustomerID: customerID,
		Name: request.Name, Email: request.Email, MetadataPlan: request.Metadata.Plan,
		MetadataCountry: request.Metadata.Country, MetadataLocation: request.Metadata.Location,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return customermodels.Customer{}, customermodels.ErrCustomerNotFound
	}
	if err != nil {
		return customermodels.Customer{}, fmt.Errorf("update customer: %w", err)
	}
	return mapCustomer(row), nil
}

func mapCustomer(row customerdb.Customer) customermodels.Customer {
	return customermodels.Customer{
		ID: row.ID, ProjectEnvironmentID: row.ProjectEnvironmentID,
		CustomerID: row.CustomerID, Name: row.Name, Email: row.Email,
		Metadata: customermodels.Metadata{
			Plan: row.MetadataPlan, Country: row.MetadataCountry, Location: row.MetadataLocation,
		},
		CreatedAt: row.CreatedAt.Time, UpdatedAt: row.UpdatedAt.Time,
	}
}

func customerExists(err error) bool {
	var databaseError *pgconn.PgError
	return errors.As(err, &databaseError) &&
		databaseError.ConstraintName == "customers_project_environment_id_customer_id_key"
}
