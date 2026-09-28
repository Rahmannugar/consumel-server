package handlers

import (
	"time"

	customermodels "github.com/Rahmannugar/consumel-server/internal/customers/models"
)

type customerResponse struct {
	CustomerID string                  `json:"customerId"`
	Name       *string                 `json:"name"`
	Email      *string                 `json:"email"`
	Metadata   customermodels.Metadata `json:"metadata"`
	CreatedAt  time.Time               `json:"createdAt"`
	UpdatedAt  time.Time               `json:"updatedAt"`
}

type customersResponse struct {
	Customers  []customerResponse `json:"customers"`
	NextCursor *string            `json:"nextCursor"`
}

func customerJSON(customer customermodels.Customer) customerResponse {
	return customerResponse{
		CustomerID: customer.CustomerID, Name: customer.Name, Email: customer.Email,
		Metadata: customer.Metadata, CreatedAt: customer.CreatedAt, UpdatedAt: customer.UpdatedAt,
	}
}

func customerListJSON(
	customers []customermodels.Customer,
	next *customermodels.ListCursor,
) customersResponse {
	result := make([]customerResponse, 0, len(customers))
	for _, customer := range customers {
		result = append(result, customerJSON(customer))
	}
	var nextCursor *string
	if next != nil {
		encoded := customermodels.EncodeCursor(*next)
		nextCursor = &encoded
	}
	return customersResponse{Customers: result, NextCursor: nextCursor}
}
