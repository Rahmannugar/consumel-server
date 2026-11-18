package models

import (
	"errors"
	"math"
	"time"

	"github.com/google/uuid"
)

var (
	ErrAllowanceInvalid       = errors.New("invalid recurring allowance")
	ErrAllowanceNotFound      = errors.New("recurring allowance not found")
	ErrAllowanceCursorInvalid = errors.New("invalid allowance cursor")
	ErrAllowanceConflict      = errors.New("recurring allowance version changed")
)

// StartsAt is immutable once created. Version zero creates; later writes require
// the latest version so two operators cannot silently overwrite a pending edit.
type AllowanceRequest struct {
	IntervalUnit  string    `json:"intervalUnit" validate:"omitempty,oneof=minute hour day week month year" enums:"minute,hour,day,week,month,year" example:"hour"`
	IntervalCount int32     `json:"intervalCount" validate:"omitempty,min=1,max=1000000" example:"6"`
	StartsAt      time.Time `json:"startsAt" validate:"required" format:"date-time" example:"2026-12-01T09:00:00Z"`
	Quantity      int64     `json:"quantity" validate:"required,min=1" format:"int64" example:"1000"`
	RolloverCap   int64     `json:"rolloverCap" validate:"min=0" format:"int64" example:"2000"`
	Enabled       bool      `json:"enabled" example:"true"`
	Version       int64     `json:"version" validate:"min=0" format:"int64" example:"0"`
}

func (r AllowanceRequest) Validate() (AllowanceRequest, error) {
	r.StartsAt = r.StartsAt.UTC()
	// Omitted schedule fields preserve the original monthly API contract.
	if r.IntervalUnit == "" && r.IntervalCount == 0 {
		r.IntervalUnit = "month"
		r.IntervalCount = 1
	}
	schedule := AllowanceSchedule{Unit: r.IntervalUnit, Count: r.IntervalCount, Anchor: r.StartsAt}
	if _, _, _, err := schedule.Period(r.StartsAt); err != nil {
		return r, ErrAllowanceInvalid
	}
	if r.StartsAt.IsZero() || r.StartsAt.Year() < 2000 || r.StartsAt.Year() > 9998 || !r.StartsAt.Equal(r.StartsAt.Truncate(time.Minute)) || r.Quantity <= 0 || r.RolloverCap < 0 || r.RolloverCap > math.MaxInt64-r.Quantity || r.Version < 0 || r.Version == math.MaxInt64 {
		return r, ErrAllowanceInvalid
	}
	return r, nil
}

type AllowanceRevision struct {
	IntervalUnit     string    `json:"intervalUnit" example:"hour"`
	IntervalCount    int32     `json:"intervalCount" example:"6"`
	ScheduleAnchorAt time.Time `json:"scheduleAnchorAt" format:"date-time" example:"2026-12-01T09:00:00Z"`
	Version          int64     `json:"version" format:"int64" example:"1"`
	EffectiveAt      time.Time `json:"effectiveAt" format:"date-time" example:"2026-12-01T09:00:00Z"`
	Quantity         int64     `json:"quantity" format:"int64" example:"1000"`
	RolloverCap      int64     `json:"rolloverCap" format:"int64" example:"2000"`
	Enabled          bool      `json:"enabled" example:"true"`
}
type Allowance struct {
	ID           uuid.UUID          `json:"id" format:"uuid" example:"019a5db9-3e67-7000-8000-000000000001"`
	StartsAt     time.Time          `json:"startsAt" format:"date-time" example:"2026-12-01T09:00:00Z"`
	NextPeriodAt time.Time          `json:"nextPeriodAt" format:"date-time" example:"2026-12-01T15:00:00Z"`
	Version      int64              `json:"version" format:"int64" example:"1"`
	Current      *AllowanceRevision `json:"current"`
	Pending      *AllowanceRevision `json:"pending"`
}

// MonthBoundary preserves the legacy monthly calendar helper.
func MonthBoundary(anchor time.Time, offset int) time.Time {
	return calendarAllowanceBoundary(anchor.UTC(), int64(offset))
}
func AllowancePeriod(anchor, at time.Time) (time.Time, time.Time, int) {
	start, end, index, _ := (AllowanceSchedule{Unit: "month", Count: 1, Anchor: anchor}).Period(at)
	return start, end, int(index)
}

// AdvanceCarry computes untouched missed periods without an outage-sized loop
// or multiplication overflow. Only the policy's own unspent grant is carried.
func AdvanceCarry(remaining, quantity, cap int64, periods int64) int64 {
	remaining = min(remaining, cap)
	if periods <= 0 || cap == 0 {
		return remaining
	}
	room := cap - remaining
	if int64(periods) > room/quantity {
		return cap
	}
	return remaining + int64(periods)*quantity
}

type AllowanceWork struct {
	BalanceID uuid.UUID
	Attempt   int
}

// Period history describes actual issuance, including bounded outage recovery.
type AllowancePeriodRecord struct {
	PeriodStart     time.Time `json:"periodStart"`
	PeriodEnd       time.Time `json:"periodEnd"`
	Revision        int64     `json:"revision"`
	Quantity        int64     `json:"quantity"`
	CarriedQuantity int64     `json:"carriedQuantity"`
	SkippedPeriods  int64     `json:"skippedPeriods"`
	SourceType      string    `json:"sourceType"`
	SourceID        uuid.UUID `json:"sourceId"`
}
type AllowancePeriods struct {
	Periods    []AllowancePeriodRecord `json:"periods"`
	NextCursor *string                 `json:"nextCursor"`
}
