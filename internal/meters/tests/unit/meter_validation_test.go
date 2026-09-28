package unit_test

import (
	"errors"
	"testing"

	metermodels "github.com/Rahmannugar/consumel-server/internal/meters/models"
)

func TestCreateMeterRequestValidation(t *testing.T) {
	description := "  Requests processed by your API.  "
	request, err := (metermodels.CreateMeterRequest{
		MeterKey: " api_calls ", Name: " API calls ", Description: &description,
		Type: metermodels.MeterTypePostpaid,
	}).Validate()
	if err != nil {
		t.Fatalf("validate meter: %v", err)
	}
	if request.MeterKey != "api_calls" || request.Name != "API calls" ||
		request.Description == nil || *request.Description != "Requests processed by your API." {
		t.Fatalf("validated request = %#v", request)
	}
}

func TestCreateMeterRequestRejectsInvalidDomainValues(t *testing.T) {
	tests := []struct {
		name    string
		request metermodels.CreateMeterRequest
		want    error
	}{
		{name: "uppercase key", request: metermodels.CreateMeterRequest{MeterKey: "API_calls", Name: "API calls", Type: metermodels.MeterTypePostpaid}, want: metermodels.ErrMeterKeyInvalid},
		{name: "missing name", request: metermodels.CreateMeterRequest{MeterKey: "api_calls", Type: metermodels.MeterTypePostpaid}, want: metermodels.ErrMeterNameInvalid},
		{name: "unknown type", request: metermodels.CreateMeterRequest{MeterKey: "api_calls", Name: "API calls", Type: "usage"}, want: metermodels.ErrMeterTypeInvalid},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := test.request.Validate()
			if !errors.Is(err, test.want) {
				t.Fatalf("validation error = %v, want %v", err, test.want)
			}
		})
	}
}
