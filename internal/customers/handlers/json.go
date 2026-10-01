package handlers

import (
	customermodels "github.com/Rahmannugar/consumel-server/internal/customers/models"
	"github.com/Rahmannugar/consumel-server/internal/openapi"
)

func metadataJSON(metadata customermodels.Metadata) openapi.CustomerMetadata {
	return openapi.CustomerMetadata{
		Plan: metadata.Plan, Country: metadata.Country, Location: metadata.Location,
	}
}

func customerJSON(customer customermodels.Customer) openapi.Customer {
	return openapi.Customer{
		CustomerID: customer.CustomerID, Name: customer.Name, Email: customer.Email,
		Metadata: metadataJSON(customer.Metadata), CreatedAt: customer.CreatedAt, UpdatedAt: customer.UpdatedAt,
	}
}

func customerListJSON(
	customers []customermodels.Customer,
	next *customermodels.ListCursor,
) openapi.Customers {
	result := make([]openapi.Customer, 0, len(customers))
	for _, customer := range customers {
		result = append(result, customerJSON(customer))
	}
	var nextCursor *string
	if next != nil {
		encoded := customermodels.EncodeCursor(*next)
		nextCursor = &encoded
	}
	return openapi.Customers{Customers: result, NextCursor: nextCursor}
}
