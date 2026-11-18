package models

import "time"

// Fixed units are elapsed UTC time. Calendar units retain the anchor's day and
// time; shorter months clamp without changing later renewal dates.
type AllowanceSchedule struct {
	Unit   string
	Count  int32
	Anchor time.Time
}

func (s AllowanceSchedule) Period(at time.Time) (time.Time, time.Time, int64, error) {
	if s.Count < 1 || s.Count > 1000000 || s.Anchor.IsZero() {
		return time.Time{}, time.Time{}, 0, ErrAllowanceInvalid
	}
	anchor := s.Anchor.UTC()
	at = at.UTC()
	if at.Before(anchor) {
		return time.Time{}, time.Time{}, 0, ErrAllowanceInvalid
	}
	var start, end time.Time
	var index int64
	switch s.Unit {
	case "minute", "hour", "day", "week":
		seconds := map[string]int64{"minute": 60, "hour": 3600, "day": 86400, "week": 604800}[s.Unit] * int64(s.Count)
		index = (at.Unix() - anchor.Unix()) / seconds
		start = time.Unix(anchor.Unix()+index*seconds, 0).UTC()
		end = time.Unix(anchor.Unix()+(index+1)*seconds, 0).UTC()
	case "month", "year":
		months := int64(s.Count)
		if s.Unit == "year" {
			months *= 12
		}
		elapsed := int64((at.Year()-anchor.Year())*12 + int(at.Month()-anchor.Month()))
		index = elapsed / months
		start = calendarAllowanceBoundary(anchor, index*months)
		if start.After(at) {
			index--
			start = calendarAllowanceBoundary(anchor, index*months)
		}
		end = calendarAllowanceBoundary(anchor, (index+1)*months)
	default:
		return time.Time{}, time.Time{}, 0, ErrAllowanceInvalid
	}
	if start.Year() < 2000 || end.Year() > 9999 || !end.After(start) {
		return time.Time{}, time.Time{}, 0, ErrAllowanceInvalid
	}
	return start, end, index, nil
}
func calendarAllowanceBoundary(anchor time.Time, months int64) time.Time {
	first := time.Date(anchor.Year(), anchor.Month()+time.Month(months), 1, anchor.Hour(), anchor.Minute(), 0, 0, time.UTC)
	last := time.Date(first.Year(), first.Month()+1, 0, 0, 0, 0, 0, time.UTC).Day()
	return time.Date(first.Year(), first.Month(), min(anchor.Day(), last), anchor.Hour(), anchor.Minute(), 0, 0, time.UTC)
}
