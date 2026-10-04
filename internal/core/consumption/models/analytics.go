package models

import (
	"errors"
	"time"
)

var ErrAnalyticsFilterInvalid = errors.New("analytics filter is invalid")

type AnalyticsInterval string

const (
	AnalyticsIntervalHour AnalyticsInterval = "hour"
	AnalyticsIntervalDay  AnalyticsInterval = "day"
)

type AnalyticsFilter struct {
	CustomerID string
	MeterKey   string
	From       time.Time
	To         time.Time
	Interval   AnalyticsInterval
}

type AnalyticsBucket struct {
	Start              time.Time
	AcceptedOperations int64
	DeniedOperations   int64
	AcceptedQuantity   int64
	DeniedQuantity     int64
	BillableOperations int64
}

type AnalyticsSummary struct {
	AcceptedOperations int64
	DeniedOperations   int64
	AcceptedQuantity   int64
	DeniedQuantity     int64
	BillableOperations int64
}

type Analytics struct {
	From     time.Time
	To       time.Time
	Interval AnalyticsInterval
	Summary  AnalyticsSummary
	Buckets  []AnalyticsBucket
}
