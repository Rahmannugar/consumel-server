package models

import (
	"errors"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
)

const (
	MaximumMeterKeyLength         = 120
	MaximumMeterNameLength        = 120
	MaximumMeterDescriptionLength = 500
	MaximumMeterSearchLength      = 120
)

var (
	ErrMeterKeyInvalid         = errors.New("meter key is invalid")
	ErrMeterNameInvalid        = errors.New("meter name is invalid")
	ErrMeterDescriptionInvalid = errors.New("meter description is invalid")
	ErrMeterTypeInvalid        = errors.New("meter type is invalid")
	ErrMeterExists             = errors.New("meter already exists")
	ErrMeterDefinitionConflict = errors.New("meter definition conflicts with the project meter")
	ErrMeterNotFound           = errors.New("meter not found")
	ErrCursorInvalid           = errors.New("meter cursor is invalid")
	ErrMeterSearchInvalid      = errors.New("meter search is invalid")
	meterKeyPattern            = regexp.MustCompile(`^[a-z][a-z0-9_-]*$`)
)

type MeterType string

const (
	MeterTypePrepaid  MeterType = "prepaid"
	MeterTypePostpaid MeterType = "postpaid"
	MeterTypeHybrid   MeterType = "hybrid"
)

type CreateMeterRequest struct {
	MeterKey    string    `json:"meterKey" validate:"required,min=1,max=120" pattern:"^[a-z][a-z0-9_-]*$" example:"api_calls"`
	Name        string    `json:"name" validate:"required,min=1,max=120" example:"API calls"`
	Description *string   `json:"description,omitempty" validate:"min=1,max=500" example:"Requests processed by your API."`
	Type        MeterType `json:"type" validate:"required" example:"postpaid"`
}

type Meter struct {
	ID                   uuid.UUID
	ProjectID            uuid.UUID
	ProjectEnvironmentID uuid.UUID
	MeterKey             string
	Name                 string
	Description          *string
	Type                 MeterType
	CreatedAt            time.Time
	UpdatedAt            time.Time
}

type ListCursor struct {
	CreatedAt time.Time
	ID        uuid.UUID
}

func (request CreateMeterRequest) Validate() (CreateMeterRequest, error) {
	request.MeterKey = strings.TrimSpace(request.MeterKey)
	request.Name = strings.TrimSpace(request.Name)
	switch {
	case !ValidMeterKey(request.MeterKey):
		return CreateMeterRequest{}, ErrMeterKeyInvalid
	case request.Name == "" || utf8.RuneCountInString(request.Name) > MaximumMeterNameLength:
		return CreateMeterRequest{}, ErrMeterNameInvalid
	case request.Type != MeterTypePrepaid && request.Type != MeterTypePostpaid && request.Type != MeterTypeHybrid:
		return CreateMeterRequest{}, ErrMeterTypeInvalid
	}
	if request.Description != nil {
		description := strings.TrimSpace(*request.Description)
		if description == "" {
			request.Description = nil
		} else if utf8.RuneCountInString(description) > MaximumMeterDescriptionLength {
			return CreateMeterRequest{}, ErrMeterDescriptionInvalid
		} else {
			request.Description = &description
		}
	}
	return request, nil
}

func ValidMeterKey(value string) bool {
	return value != "" &&
		utf8.RuneCountInString(value) <= MaximumMeterKeyLength &&
		meterKeyPattern.MatchString(value)
}
