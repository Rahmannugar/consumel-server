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
)

var (
	ErrMeterKeyInvalid         = errors.New("meter key is invalid")
	ErrMeterNameInvalid        = errors.New("meter name is invalid")
	ErrMeterDescriptionInvalid = errors.New("meter description is invalid")
	ErrMeterTypeInvalid        = errors.New("meter type is invalid")
	ErrMeterExists             = errors.New("meter already exists")
	ErrMeterNotFound           = errors.New("meter not found")
	ErrCursorInvalid           = errors.New("meter cursor is invalid")
	meterKeyPattern            = regexp.MustCompile(`^[a-z][a-z0-9_-]*$`)
)

type MeterType string

const (
	MeterTypePrepaid  MeterType = "prepaid"
	MeterTypePostpaid MeterType = "postpaid"
	MeterTypeHybrid   MeterType = "hybrid"
)

type CreateMeterRequest struct {
	MeterKey    string    `json:"meterKey"`
	Name        string    `json:"name"`
	Description *string   `json:"description,omitempty"`
	Type        MeterType `json:"type"`
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
	case request.MeterKey == "",
		utf8.RuneCountInString(request.MeterKey) > MaximumMeterKeyLength,
		!meterKeyPattern.MatchString(request.MeterKey):
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

func CreateMeterRequestOpenAPISchema() map[string]any {
	return objectSchema([]string{"meterKey", "name", "type"}, map[string]any{
		"meterKey": map[string]any{
			"type": "string", "minLength": 1, "maxLength": MaximumMeterKeyLength,
			"pattern": `^[a-z][a-z0-9_-]*$`, "example": "api_calls",
		},
		"name": map[string]any{
			"type": "string", "minLength": 1, "maxLength": MaximumMeterNameLength,
			"example": "API calls",
		},
		"description": map[string]any{
			"type": []string{"string", "null"}, "minLength": 1,
			"maxLength": MaximumMeterDescriptionLength,
			"example":   "Requests processed by your API.",
		},
		"type": map[string]any{
			"type": "string", "enum": []string{"prepaid", "postpaid", "hybrid"},
			"example": "postpaid",
		},
	})
}

func objectSchema(required []string, properties map[string]any) map[string]any {
	result := map[string]any{"type": "object", "additionalProperties": false, "properties": properties}
	if len(required) > 0 {
		result["required"] = required
	}
	return result
}
