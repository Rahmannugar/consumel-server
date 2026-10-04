package handlers

import (
	consumptionmodels "github.com/Rahmannugar/consumel-server/internal/core/consumption/models"
	"github.com/Rahmannugar/consumel-server/internal/openapi"
)

func balanceJSON(balance consumptionmodels.Balance) openapi.Balance {
	return openapi.Balance{
		ID: balance.ID, CustomerID: balance.CustomerID, MeterKey: balance.MeterKey,
		Quantity: balance.Quantity, NextExpiresAt: balance.NextExpiresAt,
		CreatedAt: balance.CreatedAt, UpdatedAt: balance.UpdatedAt,
	}
}

func balanceListJSON(balances []consumptionmodels.Balance) openapi.Balances {
	result := make([]openapi.Balance, 0, len(balances))
	for _, balance := range balances {
		result = append(result, balanceJSON(balance))
	}
	return openapi.Balances{Balances: result}
}

func entitlementGrantListJSON(grants []consumptionmodels.EntitlementGrant, next *consumptionmodels.OperationListCursor) openapi.EntitlementGrants {
	result := make([]openapi.EntitlementGrant, 0, len(grants))
	for _, grant := range grants {
		result = append(result, openapi.EntitlementGrant{
			ID: grant.ID, GrantedQuantity: grant.GrantedQuantity,
			RemainingQuantity: grant.RemainingQuantity, Status: string(grant.Status),
			ExpiresAt: grant.ExpiresAt, CreatedAt: grant.CreatedAt,
		})
	}
	var encoded *string
	if next != nil {
		value := consumptionmodels.EncodeOperationCursor(*next)
		encoded = &value
	}
	return openapi.EntitlementGrants{Grants: result, NextCursor: encoded}
}

func balanceActivityListJSON(activity []consumptionmodels.BalanceActivity, next *consumptionmodels.OperationListCursor) openapi.BalanceActivityList {
	result := make([]openapi.BalanceActivity, 0, len(activity))
	for _, item := range activity {
		result = append(result, openapi.BalanceActivity{
			ID: item.ID, Kind: item.Kind, QuantityChange: item.QuantityChange,
			ResultingQuantity: item.ResultingQuantity, ExpiresAt: item.ExpiresAt,
			SourceType: item.SourceType, OccurredAt: item.OccurredAt,
		})
	}
	var encoded *string
	if next != nil {
		value := consumptionmodels.EncodeOperationCursor(*next)
		encoded = &value
	}
	return openapi.BalanceActivityList{Activity: result, NextCursor: encoded}
}

func usageEventJSON(event consumptionmodels.UsageEvent) openapi.UsageEvent {
	return openapi.UsageEvent{
		ID: event.ID, CustomerID: event.CustomerID, MeterKey: event.MeterKey,
		Quantity: event.Quantity, MeterType: string(event.MeterType),
		BalanceDebited: event.BalanceDebited, RemainingBalance: event.RemainingBalance,
		Billable: event.Billable, CreatedAt: event.CreatedAt,
	}
}

func operationListJSON(
	operations []consumptionmodels.Operation,
	next *consumptionmodels.OperationListCursor,
) openapi.UsageOperations {
	result := make([]openapi.UsageOperation, 0, len(operations))
	for _, operation := range operations {
		result = append(result, operationJSON(operation))
	}
	var encoded *string
	if next != nil {
		value := consumptionmodels.EncodeOperationCursor(*next)
		encoded = &value
	}
	return openapi.UsageOperations{Operations: result, NextCursor: encoded}
}

func operationJSON(operation consumptionmodels.Operation) openapi.UsageOperation {
	return openapi.UsageOperation{
		ID: operation.ID, CustomerID: operation.CustomerID,
		MeterKey: operation.MeterKey, Quantity: operation.Quantity,
		MeterType: string(operation.MeterType), Status: string(operation.Status),
		DenialReason: operation.DenialReason, BalanceDebited: operation.BalanceDebited,
		RemainingBalance: operation.RemainingBalance, Billable: operation.Billable,
		ReplayCount: operation.ReplayCount, LastReplayedAt: operation.LastReplayedAt,
		CreatedAt: operation.CreatedAt,
	}
}

func analyticsJSON(analytics consumptionmodels.Analytics) openapi.UsageAnalytics {
	buckets := make([]openapi.UsageAnalyticsBucket, 0, len(analytics.Buckets))
	for _, bucket := range analytics.Buckets {
		buckets = append(buckets, openapi.UsageAnalyticsBucket{
			Start: bucket.Start, AcceptedOperations: bucket.AcceptedOperations,
			DeniedOperations: bucket.DeniedOperations, AcceptedQuantity: bucket.AcceptedQuantity,
			DeniedQuantity: bucket.DeniedQuantity, BillableOperations: bucket.BillableOperations,
		})
	}
	return openapi.UsageAnalytics{
		From: analytics.From, To: analytics.To, Interval: string(analytics.Interval),
		Summary: openapi.UsageAnalyticsSummary{
			AcceptedOperations: analytics.Summary.AcceptedOperations,
			DeniedOperations:   analytics.Summary.DeniedOperations,
			AcceptedQuantity:   analytics.Summary.AcceptedQuantity,
			DeniedQuantity:     analytics.Summary.DeniedQuantity,
			BillableOperations: analytics.Summary.BillableOperations,
		},
		Buckets: buckets,
	}
}
