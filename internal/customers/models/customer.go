package models

import (
	"errors"
	"net/mail"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
)

const (
	MaximumCustomerIDLength     = 255
	MaximumCustomerNameLength   = 200
	MaximumCustomerEmailLength  = 320
	MaximumMetadataValueLength  = 120
	MaximumCustomerSearchLength = 120
)

var (
	ErrCustomerIDRequired      = errors.New("customer ID is required")
	ErrCustomerIDTooLong       = errors.New("customer ID is too long")
	ErrCustomerNameInvalid     = errors.New("customer name is invalid")
	ErrCustomerEmailInvalid    = errors.New("customer email is invalid")
	ErrCustomerMetadataInvalid = errors.New("customer metadata is invalid")
	ErrCustomerExists          = errors.New("customer already exists")
	ErrCustomerNotFound        = errors.New("customer not found")
	ErrCursorInvalid           = errors.New("customer cursor is invalid")
	ErrCustomerSearchInvalid   = errors.New("customer search is invalid")
)

type Metadata struct {
	Plan     *string `json:"plan,omitempty" validate:"min=1,max=120" example:"growth"`
	Country  *string `json:"country,omitempty" pattern:"^[A-Za-z]{2}$" example:"US"`
	Location *string `json:"location,omitempty" validate:"min=1,max=120" example:"New York, NY"`
}

type CreateCustomerRequest struct {
	CustomerID string   `json:"customerId" validate:"required,min=1,max=255" example:"user_123"`
	Name       *string  `json:"name,omitempty" validate:"min=1,max=200" example:"Jordan Lee"`
	Email      *string  `json:"email,omitempty" validate:"max=320" format:"email" example:"jordan@example.com"`
	Metadata   Metadata `json:"metadata"`
}

type UpdateCustomerRequest struct {
	Name     *string  `json:"name,omitempty" validate:"min=1,max=200" example:"Jordan Lee"`
	Email    *string  `json:"email,omitempty" validate:"max=320" format:"email" example:"jordan@example.com"`
	Metadata Metadata `json:"metadata"`
}

type Customer struct {
	ID                   uuid.UUID
	ProjectEnvironmentID uuid.UUID
	CustomerID           string
	Name                 *string
	Email                *string
	Metadata             Metadata
	CreatedAt            time.Time
	UpdatedAt            time.Time
}

type ListCursor struct {
	CreatedAt time.Time
	ID        uuid.UUID
}

func (request CreateCustomerRequest) Validate() (CreateCustomerRequest, error) {
	request.CustomerID = strings.TrimSpace(request.CustomerID)
	switch {
	case request.CustomerID == "":
		return CreateCustomerRequest{}, ErrCustomerIDRequired
	case utf8.RuneCountInString(request.CustomerID) > MaximumCustomerIDLength:
		return CreateCustomerRequest{}, ErrCustomerIDTooLong
	}
	fields, err := validateFields(request.Name, request.Email, request.Metadata)
	if err != nil {
		return CreateCustomerRequest{}, err
	}
	request.Name, request.Email, request.Metadata = fields.Name, fields.Email, fields.Metadata
	return request, nil
}

func (request UpdateCustomerRequest) Validate() (UpdateCustomerRequest, error) {
	fields, err := validateFields(request.Name, request.Email, request.Metadata)
	if err != nil {
		return UpdateCustomerRequest{}, err
	}
	return fields, nil
}

func validateFields(name, email *string, metadata Metadata) (UpdateCustomerRequest, error) {
	name, err := optionalText(name, MaximumCustomerNameLength, ErrCustomerNameInvalid)
	if err != nil {
		return UpdateCustomerRequest{}, err
	}
	email, err = optionalText(email, MaximumCustomerEmailLength, ErrCustomerEmailInvalid)
	if err != nil {
		return UpdateCustomerRequest{}, err
	}
	if email != nil {
		parsed, parseErr := mail.ParseAddress(*email)
		if parseErr != nil || parsed.Address != *email {
			return UpdateCustomerRequest{}, ErrCustomerEmailInvalid
		}
	}
	metadata.Plan, err = optionalText(metadata.Plan, MaximumMetadataValueLength, ErrCustomerMetadataInvalid)
	if err != nil {
		return UpdateCustomerRequest{}, err
	}
	metadata.Country, err = normalizeCountryCode(metadata.Country)
	if err != nil {
		return UpdateCustomerRequest{}, err
	}
	metadata.Location, err = optionalText(metadata.Location, MaximumMetadataValueLength, ErrCustomerMetadataInvalid)
	if err != nil {
		return UpdateCustomerRequest{}, err
	}
	return UpdateCustomerRequest{Name: name, Email: email, Metadata: metadata}, nil
}

func optionalText(value *string, maximum int, invalid error) (*string, error) {
	if value == nil {
		return nil, nil
	}
	trimmed := strings.TrimSpace(*value)
	if trimmed == "" || utf8.RuneCountInString(trimmed) > maximum {
		return nil, invalid
	}
	return &trimmed, nil
}
