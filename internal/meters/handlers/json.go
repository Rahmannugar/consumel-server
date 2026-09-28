package handlers

import (
	"time"

	metermodels "github.com/Rahmannugar/consumel-server/internal/meters/models"
)

type meterResponse struct {
	ID          string                `json:"id"`
	MeterKey    string                `json:"meterKey"`
	Name        string                `json:"name"`
	Description *string               `json:"description"`
	Type        metermodels.MeterType `json:"type"`
	CreatedAt   time.Time             `json:"createdAt"`
	UpdatedAt   time.Time             `json:"updatedAt"`
}

type metersResponse struct {
	Meters     []meterResponse `json:"meters"`
	NextCursor *string         `json:"nextCursor"`
}

func meterJSON(meter metermodels.Meter) meterResponse {
	return meterResponse{
		ID: meter.ID.String(), MeterKey: meter.MeterKey, Name: meter.Name,
		Description: meter.Description, Type: meter.Type,
		CreatedAt: meter.CreatedAt, UpdatedAt: meter.UpdatedAt,
	}
}

func meterListJSON(meters []metermodels.Meter, next *metermodels.ListCursor) metersResponse {
	result := make([]meterResponse, 0, len(meters))
	for _, meter := range meters {
		result = append(result, meterJSON(meter))
	}
	var nextCursor *string
	if next != nil {
		encoded := metermodels.EncodeCursor(*next)
		nextCursor = &encoded
	}
	return metersResponse{Meters: result, NextCursor: nextCursor}
}
