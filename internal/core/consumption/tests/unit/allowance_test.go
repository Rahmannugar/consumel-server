package unit_test

import (
	"errors"
	"math"
	"testing"
	"time"

	consumptionmodels "github.com/Rahmannugar/consumel-server/internal/core/consumption/models"
)

func TestAllowanceMonthEndDoesNotDrift(t *testing.T) {
	anchor := time.Date(2024, 1, 31, 0, 0, 0, 0, time.UTC)
	cases := []struct {
		at    string
		start string
		end   string
	}{
		{"2024-02-28T23:59:59Z", "2024-01-31T00:00:00Z", "2024-02-29T00:00:00Z"},
		{"2024-02-29T00:00:00Z", "2024-02-29T00:00:00Z", "2024-03-31T00:00:00Z"},
		{"2025-02-28T00:00:00Z", "2025-02-28T00:00:00Z", "2025-03-31T00:00:00Z"},
	}
	for _, tc := range cases {
		t.Run(tc.at, func(t *testing.T) {
			at, err := time.Parse(time.RFC3339, tc.at)
			if err != nil {
				t.Fatal(err)
			}
			start, end, _ := consumptionmodels.AllowancePeriod(anchor, at)
			if start.Format(time.RFC3339) != tc.start || end.Format(time.RFC3339) != tc.end {
				t.Fatalf("period = %s to %s", start, end)
			}
		})
	}
}
func TestAllowanceCarrySaturatesWithoutOverflow(t *testing.T) {
	for _, tc := range []struct {
		remaining, quantity, cap int64
		missed                   int64
		want                     int64
	}{
		{60, 100, 150, 0, 60}, {60, 100, 150, 3, 150}, {60, 100, 0, 5, 0},
		{0, math.MaxInt64, math.MaxInt64, 100000, math.MaxInt64},
		{1, 2, 100, 4, 9},
	} {
		if got := consumptionmodels.AdvanceCarry(tc.remaining, tc.quantity, tc.cap, tc.missed); got != tc.want {
			t.Fatalf("carry %v = %d", tc, got)
		}
	}
}
func TestAllowanceRejectsInvalidScheduleAndCapacity(t *testing.T) {
	valid := consumptionmodels.AllowanceRequest{StartsAt: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), Quantity: 100, Enabled: true}
	for _, change := range []func(*consumptionmodels.AllowanceRequest){
		func(r *consumptionmodels.AllowanceRequest) { r.StartsAt = r.StartsAt.Add(time.Second) },
		func(r *consumptionmodels.AllowanceRequest) { r.Quantity = 0 },
		func(r *consumptionmodels.AllowanceRequest) { r.RolloverCap = math.MaxInt64 },
		func(r *consumptionmodels.AllowanceRequest) { r.Version = -1 },
	} {
		r := valid
		change(&r)
		if _, err := r.Validate(); !errors.Is(err, consumptionmodels.ErrAllowanceInvalid) {
			t.Fatalf("invalid request accepted: %#v: %v", r, err)
		}
	}
}

func TestCustomAllowancePeriods(t *testing.T) {
	for _, tc := range []struct {
		unit                   string
		count                  int32
		anchor, at, start, end string
	}{
		{"minute", 30, "2026-01-01T09:15:00Z", "2026-01-01T10:12:00Z", "2026-01-01T09:45:00Z", "2026-01-01T10:15:00Z"},
		{"hour", 6, "2026-01-01T09:00:00Z", "2026-01-02T02:00:00Z", "2026-01-01T21:00:00Z", "2026-01-02T03:00:00Z"},
		{"day", 2, "2026-01-01T09:00:00Z", "2026-01-05T08:00:00Z", "2026-01-03T09:00:00Z", "2026-01-05T09:00:00Z"},
		{"week", 2, "2026-01-01T09:00:00Z", "2026-01-15T09:00:00Z", "2026-01-15T09:00:00Z", "2026-01-29T09:00:00Z"},
		{"month", 3, "2026-01-31T09:15:00Z", "2026-04-30T09:15:00Z", "2026-04-30T09:15:00Z", "2026-07-31T09:15:00Z"},
		{"year", 1, "2024-02-29T09:00:00Z", "2025-02-28T09:00:00Z", "2025-02-28T09:00:00Z", "2026-02-28T09:00:00Z"},
		{"year", 1, "2024-02-29T09:00:00Z", "2028-02-29T09:00:00Z", "2028-02-29T09:00:00Z", "2029-02-28T09:00:00Z"},
	} {
		t.Run(tc.unit+tc.at, func(t *testing.T) {
			parse := func(value string) time.Time {
				at, err := time.Parse(time.RFC3339, value)
				if err != nil {
					t.Fatal(err)
				}
				return at
			}
			start, end, _, err := (consumptionmodels.AllowanceSchedule{Unit: tc.unit, Count: tc.count, Anchor: parse(tc.anchor)}).Period(parse(tc.at))
			if err != nil || !start.Equal(parse(tc.start)) || !end.Equal(parse(tc.end)) {
				t.Fatalf("period=%s to %s, %v", start, end, err)
			}
		})
	}
}
func TestMinuteRecoveryExceedsInt32WithoutLooping(t *testing.T) {
	anchor := time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)
	at := time.Date(9998, 1, 1, 0, 0, 0, 0, time.UTC)
	start, end, index, err := (consumptionmodels.AllowanceSchedule{Unit: "minute", Count: 1, Anchor: anchor}).Period(at)
	if err != nil || index <= math.MaxInt32 || !start.Equal(at) || end.Sub(start) != time.Minute {
		t.Fatalf("period=%s to %s,index=%d,error=%v", start, end, index, err)
	}
	if got := consumptionmodels.AdvanceCarry(20, 100, 1000, index); got != 1000 {
		t.Fatalf("bounded carry=%d", got)
	}
}
func TestScheduleValidationPreservesLegacyMonthlyRequests(t *testing.T) {
	request := consumptionmodels.AllowanceRequest{StartsAt: time.Date(2026, 1, 1, 9, 30, 0, 0, time.UTC), Quantity: 10}
	normalized, err := request.Validate()
	if err != nil || normalized.IntervalUnit != "month" || normalized.IntervalCount != 1 {
		t.Fatalf("legacy request=%#v,%v", normalized, err)
	}
	for _, unit := range []string{"minute", "hour", "day", "week", "month", "year"} {
		request.IntervalUnit = unit
		request.IntervalCount = 1
		if _, err = request.Validate(); err != nil {
			t.Fatalf("unit %s: %v", unit, err)
		}
	}
	request.IntervalUnit = "second"
	if _, err = request.Validate(); !errors.Is(err, consumptionmodels.ErrAllowanceInvalid) {
		t.Fatalf("seconds accepted: %v", err)
	}
	request.IntervalUnit = "minute"
	request.IntervalCount = 0
	if _, err = request.Validate(); !errors.Is(err, consumptionmodels.ErrAllowanceInvalid) {
		t.Fatalf("zero count accepted: %v", err)
	}
}
