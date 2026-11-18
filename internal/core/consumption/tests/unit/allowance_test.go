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
		missed                   int
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
		func(r *consumptionmodels.AllowanceRequest) { r.StartsAt = r.StartsAt.Add(time.Hour) },
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
