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
	StartsAt    time.Time `json:"startsAt" validate:"required" format:"date-time" example:"2026-12-01T00:00:00Z"`
	Quantity    int64     `json:"quantity" validate:"required,min=1" format:"int64" example:"1000"`
	RolloverCap int64     `json:"rolloverCap" validate:"min=0" format:"int64" example:"2000"`
	Enabled     bool      `json:"enabled" example:"true"`
	Version     int64     `json:"version" validate:"min=0" format:"int64" example:"0"`
}

func (r AllowanceRequest) Validate() (AllowanceRequest, error) {
	r.StartsAt = r.StartsAt.UTC()
	if r.StartsAt.IsZero() || r.StartsAt.Year() < 2000 || r.StartsAt.Year() > 9998 || !r.StartsAt.Equal(r.StartsAt.Truncate(24*time.Hour)) || r.Quantity <= 0 || r.RolloverCap < 0 || r.RolloverCap > math.MaxInt64-r.Quantity || r.Version < 0 || r.Version == math.MaxInt64 {
		return r, ErrAllowanceInvalid
	}
	return r, nil
}

type AllowanceRevision struct {
	Version     int64     `json:"version" format:"int64" example:"1"`
	EffectiveAt time.Time `json:"effectiveAt" format:"date-time" example:"2026-12-01T00:00:00Z"`
	Quantity    int64     `json:"quantity" format:"int64" example:"1000"`
	RolloverCap int64     `json:"rolloverCap" format:"int64" example:"2000"`
	Enabled     bool      `json:"enabled" example:"true"`
}
type Allowance struct {
	ID           uuid.UUID          `json:"id" format:"uuid" example:"019a5db9-3e67-7000-8000-000000000001"`
	StartsAt     time.Time          `json:"startsAt" format:"date-time" example:"2026-12-01T00:00:00Z"`
	NextPeriodAt time.Time          `json:"nextPeriodAt" format:"date-time" example:"2027-01-01T00:00:00Z"`
	Version      int64              `json:"version" format:"int64" example:"1"`
	Current      *AllowanceRevision `json:"current"`
	Pending      *AllowanceRevision `json:"pending"`
}

// MonthBoundary always uses the original anchor day, so Jan 31 clamps to
// February's last day and returns to March 31 instead of drifting to March 28.
func MonthBoundary(anchor time.Time, offset int) time.Time {
	anchor = anchor.UTC()
	month := time.Date(anchor.Year(), anchor.Month()+time.Month(offset), 1, 0, 0, 0, 0, time.UTC)
	last := month.AddDate(0, 1, -1).Day()
	return time.Date(month.Year(), month.Month(), min(anchor.Day(), last), 0, 0, 0, 0, time.UTC)
}
func AllowancePeriod(anchor, at time.Time) (time.Time, time.Time, int) {
	anchor = anchor.UTC()
	at = at.UTC()
	months := (at.Year()-anchor.Year())*12 + int(at.Month()-anchor.Month())
	if MonthBoundary(anchor, months).After(at) {
		months--
	}
	return MonthBoundary(anchor, months), MonthBoundary(anchor, months+1), months
}

// AdvanceCarry computes untouched missed periods without an outage-sized loop
// or multiplication overflow. Only the policy's own unspent grant is carried.
func AdvanceCarry(remaining, quantity, cap int64, periods int) int64 {
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
	SkippedPeriods  int32     `json:"skippedPeriods"`
	SourceType      string    `json:"sourceType"`
	SourceID        uuid.UUID `json:"sourceId"`
}
type AllowancePeriods struct {
	Periods    []AllowancePeriodRecord `json:"periods"`
	NextCursor *string                 `json:"nextCursor"`
}
