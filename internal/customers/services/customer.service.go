package services

import (
	"context"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/Rahmannugar/consumel-server/internal/common/ids"
	customermodels "github.com/Rahmannugar/consumel-server/internal/customers/models"
	"github.com/google/uuid"
)

const (
	DefaultPageSize = 25
	MaximumPageSize = 100
)

type CustomerRepository interface {
	Create(context.Context, customermodels.Customer) (customermodels.Customer, error)
	Get(context.Context, uuid.UUID, string) (customermodels.Customer, error)
	List(context.Context, uuid.UUID, *customermodels.ListCursor, int, string) ([]customermodels.Customer, *customermodels.ListCursor, error)
	Update(context.Context, uuid.UUID, string, customermodels.UpdateCustomerRequest) (customermodels.Customer, error)
}

type CustomerService struct {
	repository CustomerRepository
}

func NewCustomerService(repository CustomerRepository) *CustomerService {
	return &CustomerService{repository: repository}
}

func (service *CustomerService) Create(
	ctx context.Context,
	projectEnvironmentID uuid.UUID,
	request customermodels.CreateCustomerRequest,
) (customermodels.Customer, error) {
	request, err := request.Validate()
	if err != nil {
		return customermodels.Customer{}, err
	}
	id, err := ids.New()
	if err != nil {
		return customermodels.Customer{}, fmt.Errorf("generate customer ID: %w", err)
	}
	return service.repository.Create(ctx, customermodels.Customer{
		ID: id, ProjectEnvironmentID: projectEnvironmentID, CustomerID: request.CustomerID,
		Name: request.Name, Email: request.Email, Metadata: request.Metadata,
	})
}

func (service *CustomerService) Get(
	ctx context.Context,
	projectEnvironmentID uuid.UUID,
	customerID string,
) (customermodels.Customer, error) {
	customerID, err := validateCustomerID(customerID)
	if err != nil {
		return customermodels.Customer{}, err
	}
	return service.repository.Get(ctx, projectEnvironmentID, customerID)
}

func (service *CustomerService) List(
	ctx context.Context,
	projectEnvironmentID uuid.UUID,
	cursor *customermodels.ListCursor,
	limit int,
	search string,
) ([]customermodels.Customer, *customermodels.ListCursor, error) {
	if limit <= 0 {
		limit = DefaultPageSize
	}
	if limit > MaximumPageSize {
		limit = MaximumPageSize
	}
	search = strings.TrimSpace(search)
	if utf8.RuneCountInString(search) > customermodels.MaximumCustomerSearchLength {
		return nil, nil, customermodels.ErrCustomerSearchInvalid
	}
	return service.repository.List(ctx, projectEnvironmentID, cursor, limit, search)
}

func (service *CustomerService) Update(
	ctx context.Context,
	projectEnvironmentID uuid.UUID,
	customerID string,
	request customermodels.UpdateCustomerRequest,
) (customermodels.Customer, error) {
	customerID, err := validateCustomerID(customerID)
	if err != nil {
		return customermodels.Customer{}, err
	}
	request, err = request.Validate()
	if err != nil {
		return customermodels.Customer{}, err
	}
	return service.repository.Update(ctx, projectEnvironmentID, customerID, request)
}

func validateCustomerID(value string) (string, error) {
	request, err := (customermodels.CreateCustomerRequest{CustomerID: strings.TrimSpace(value)}).Validate()
	if err != nil {
		return "", err
	}
	return request.CustomerID, nil
}
