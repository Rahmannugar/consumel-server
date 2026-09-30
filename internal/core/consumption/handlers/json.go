package handlers

import (
	"time"

	consumptionmodels "github.com/Rahmannugar/consumel-server/internal/core/consumption/models"
)

type balanceResponse struct {
	ID            string     `json:"id"`
	CustomerID    string     `json:"customerId"`
	MeterKey      string     `json:"meterKey"`
	Quantity      int64      `json:"quantity"`
	NextExpiresAt *time.Time `json:"nextExpiresAt"`
	CreatedAt     time.Time  `json:"createdAt"`
	UpdatedAt     time.Time  `json:"updatedAt"`
}

type balanceListResponse struct {
	Balances []balanceResponse `json:"balances"`
}

type usageEventResponse struct {
	ID               string    `json:"id"`
	CustomerID       string    `json:"customerId"`
	MeterKey         string    `json:"meterKey"`
	Quantity         int64     `json:"quantity"`
	MeterType        string    `json:"meterType"`
	BalanceDebited   int64     `json:"balanceDebited"`
	RemainingBalance *int64    `json:"remainingBalance"`
	Billable         bool      `json:"billable"`
	CreatedAt        time.Time `json:"createdAt"`
}

type operationResponse struct {
	ID               string     `json:"id"`
	CustomerID       string     `json:"customerId"`
	MeterKey         string     `json:"meterKey"`
	Quantity         int64      `json:"quantity"`
	MeterType        string     `json:"meterType"`
	Status           string     `json:"status"`
	DenialReason     *string    `json:"denialReason"`
	BalanceDebited   int64      `json:"balanceDebited"`
	RemainingBalance *int64     `json:"remainingBalance"`
	Billable         bool       `json:"billable"`
	ReplayCount      int64      `json:"replayCount"`
	LastReplayedAt   *time.Time `json:"lastReplayedAt"`
	CreatedAt        time.Time  `json:"createdAt"`
}

type operationListResponse struct {
	Operations []operationResponse `json:"operations"`
	NextCursor *string             `json:"nextCursor"`
}

func balanceJSON(balance consumptionmodels.Balance) balanceResponse {
	return balanceResponse{
		ID: balance.ID.String(), CustomerID: balance.CustomerID, MeterKey: balance.MeterKey,
		Quantity: balance.Quantity, NextExpiresAt: balance.NextExpiresAt,
		CreatedAt: balance.CreatedAt, UpdatedAt: balance.UpdatedAt,
	}
}

func balanceListJSON(balances []consumptionmodels.Balance) balanceListResponse {
	result := make([]balanceResponse, 0, len(balances))
	for _, balance := range balances {
		result = append(result, balanceJSON(balance))
	}
	return balanceListResponse{Balances: result}
}

func usageEventJSON(event consumptionmodels.UsageEvent) usageEventResponse {
	return usageEventResponse{
		ID: event.ID.String(), CustomerID: event.CustomerID, MeterKey: event.MeterKey,
		Quantity: event.Quantity, MeterType: string(event.MeterType),
		BalanceDebited: event.BalanceDebited, RemainingBalance: event.RemainingBalance,
		Billable: event.Billable, CreatedAt: event.CreatedAt,
	}
}

func operationListJSON(
	operations []consumptionmodels.Operation,
	next *consumptionmodels.OperationListCursor,
) operationListResponse {
	result := make([]operationResponse, 0, len(operations))
	for _, operation := range operations {
		result = append(result, operationJSON(operation))
	}
	var encoded *string
	if next != nil {
		value := consumptionmodels.EncodeOperationCursor(*next)
		encoded = &value
	}
	return operationListResponse{Operations: result, NextCursor: encoded}
}

func operationJSON(operation consumptionmodels.Operation) operationResponse {
	return operationResponse{
		ID: operation.ID.String(), CustomerID: operation.CustomerID,
		MeterKey: operation.MeterKey, Quantity: operation.Quantity,
		MeterType: string(operation.MeterType), Status: string(operation.Status),
		DenialReason: operation.DenialReason, BalanceDebited: operation.BalanceDebited,
		RemainingBalance: operation.RemainingBalance, Billable: operation.Billable,
		ReplayCount: operation.ReplayCount, LastReplayedAt: operation.LastReplayedAt,
		CreatedAt: operation.CreatedAt,
	}
}
