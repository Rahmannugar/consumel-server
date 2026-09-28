package unit_test

import (
	"errors"
	"testing"

	customermodels "github.com/Rahmannugar/consumel-server/internal/customers/models"
)

func TestCustomerValidationNormalizesSupportedCountryCode(t *testing.T) {
	country := " us "
	request, err := (customermodels.CreateCustomerRequest{
		CustomerID: "user_123",
		Metadata:   customermodels.Metadata{Country: &country},
	}).Validate()
	if err != nil {
		t.Fatalf("validate customer: %v", err)
	}
	if request.Metadata.Country == nil || *request.Metadata.Country != "US" {
		t.Fatalf("country = %v, want US", request.Metadata.Country)
	}
}

func TestCustomerValidationRejectsUnsupportedCountry(t *testing.T) {
	country := "United States"
	_, err := (customermodels.CreateCustomerRequest{
		CustomerID: "user_123",
		Metadata:   customermodels.Metadata{Country: &country},
	}).Validate()
	if !errors.Is(err, customermodels.ErrCustomerMetadataInvalid) {
		t.Fatalf("error = %v, want invalid metadata", err)
	}
}
