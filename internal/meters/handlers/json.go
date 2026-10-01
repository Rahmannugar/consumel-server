package handlers

import (
	metermodels "github.com/Rahmannugar/consumel-server/internal/meters/models"
	"github.com/Rahmannugar/consumel-server/internal/openapi"
)

func meterJSON(meter metermodels.Meter) openapi.Meter {
	return openapi.Meter{
		ID: meter.ID, MeterKey: meter.MeterKey, Name: meter.Name,
		Description: meter.Description, Type: string(meter.Type),
		CreatedAt: meter.CreatedAt, UpdatedAt: meter.UpdatedAt,
	}
}

func meterListJSON(meters []metermodels.Meter, next *metermodels.ListCursor) openapi.Meters {
	result := make([]openapi.Meter, 0, len(meters))
	for _, meter := range meters {
		result = append(result, meterJSON(meter))
	}
	var nextCursor *string
	if next != nil {
		encoded := metermodels.EncodeCursor(*next)
		nextCursor = &encoded
	}
	return openapi.Meters{Meters: result, NextCursor: nextCursor}
}
