package unit_test

import (
	"errors"
	"testing"

	consumptionmodels "github.com/Rahmannugar/consumel-server/internal/core/consumption/models"
)

func TestAddBalanceRequestValidationNormalizesPublicKeys(t *testing.T) {
	request, err := (consumptionmodels.AddBalanceRequest{
		CustomerID: " customer_123 ", MeterKey: " api_calls ", Quantity: 25,
	}).Validate()
	if err != nil {
		t.Fatalf("validate request: %v", err)
	}
	if request.CustomerID != "customer_123" || request.MeterKey != "api_calls" {
		t.Fatalf("normalized request = %#v", request)
	}
}

func TestAddBalanceRequestRejectsInvalidMeterKeyAndQuantity(t *testing.T) {
	tests := []struct {
		name    string
		request consumptionmodels.AddBalanceRequest
		want    error
	}{
		{name: "meter key", request: consumptionmodels.AddBalanceRequest{CustomerID: "customer_123", MeterKey: "API Calls", Quantity: 1}, want: consumptionmodels.ErrBalanceMeterInvalid},
		{name: "zero addition", request: consumptionmodels.AddBalanceRequest{CustomerID: "customer_123", MeterKey: "api_calls"}, want: consumptionmodels.ErrBalanceQuantityInvalid},
		{name: "negative addition", request: consumptionmodels.AddBalanceRequest{CustomerID: "customer_123", MeterKey: "api_calls", Quantity: -1}, want: consumptionmodels.ErrBalanceQuantityInvalid},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := test.request.Validate(); !errors.Is(err, test.want) {
				t.Fatalf("validation error = %v, want %v", err, test.want)
			}
		})
	}
}

func TestSetBalanceRequestAllowsZeroButRejectsNegativeQuantity(t *testing.T) {
	if _, err := (consumptionmodels.SetBalanceRequest{Quantity: 0}).Validate(); err != nil {
		t.Fatalf("validate zero balance: %v", err)
	}
	if _, err := (consumptionmodels.SetBalanceRequest{Quantity: -1}).Validate(); !errors.Is(err, consumptionmodels.ErrBalanceQuantityInvalid) {
		t.Fatalf("negative validation error = %v", err)
	}
}

func TestConsumeRequestValidationNormalizesKeysAndRequiresPositiveQuantity(t *testing.T) {
	request, err := (consumptionmodels.ConsumeRequest{
		CustomerID: " customer_123 ", MeterKey: " api_calls ", Quantity: 5,
	}).Validate()
	if err != nil {
		t.Fatalf("validate consume request: %v", err)
	}
	if request.CustomerID != "customer_123" || request.MeterKey != "api_calls" {
		t.Fatalf("normalized consume request = %#v", request)
	}
	if _, err := (consumptionmodels.ConsumeRequest{
		CustomerID: "customer_123", MeterKey: "api_calls", Quantity: 0,
	}).Validate(); !errors.Is(err, consumptionmodels.ErrConsumeQuantityInvalid) {
		t.Fatalf("zero quantity error = %v", err)
	}
}
