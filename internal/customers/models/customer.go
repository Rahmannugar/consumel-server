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
	Plan     *string `json:"plan,omitempty"`
	Country  *string `json:"country,omitempty"`
	Location *string `json:"location,omitempty"`
}

type CreateCustomerRequest struct {
	CustomerID string   `json:"customerId"`
	Name       *string  `json:"name,omitempty"`
	Email      *string  `json:"email,omitempty"`
	Metadata   Metadata `json:"metadata"`
}

type UpdateCustomerRequest struct {
	Name     *string  `json:"name,omitempty"`
	Email    *string  `json:"email,omitempty"`
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

func CreateCustomerRequestOpenAPISchema() map[string]any {
	properties := customerFieldSchemas()
	properties["customerId"] = map[string]any{
		"type": "string", "minLength": 1, "maxLength": MaximumCustomerIDLength,
		"example": "user_123",
	}
	return objectSchema([]string{"customerId"}, properties)
}

func UpdateCustomerRequestOpenAPISchema() map[string]any {
	return objectSchema(nil, customerFieldSchemas())
}

func customerFieldSchemas() map[string]any {
	optionalString := func(maximum int) map[string]any {
		return map[string]any{"type": []string{"string", "null"}, "minLength": 1, "maxLength": maximum}
	}
	return map[string]any{
		"name":  optionalString(MaximumCustomerNameLength),
		"email": map[string]any{"type": []string{"string", "null"}, "format": "email", "maxLength": MaximumCustomerEmailLength},
		"metadata": objectSchema(nil, map[string]any{
			"plan":     optionalString(MaximumMetadataValueLength),
			"country":  map[string]any{"type": []string{"string", "null"}, "pattern": `^[A-Za-z]{2}$`, "example": "US"},
			"location": optionalString(MaximumMetadataValueLength),
		}),
	}
}

func objectSchema(required []string, properties map[string]any) map[string]any {
	result := map[string]any{"type": "object", "additionalProperties": false, "properties": properties}
	if len(required) > 0 {
		result["required"] = required
	}
	return result
}
