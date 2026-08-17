package config_test

import (
	"scenic-ticket/config"
	"testing"
	"time"
)

func TestDateKeyUsesBusinessCalendarDay(t *testing.T) {
	beforeMidnight := time.Date(2026, 8, 17, 15, 59, 59, 0, time.UTC)
	atMidnight := time.Date(2026, 8, 17, 16, 0, 0, 0, time.UTC)

	if got := config.DateKey(beforeMidnight); got != time.Date(2026, 8, 17, 0, 0, 0, 0, time.UTC) {
		t.Fatalf("date key before business midnight = %s", got)
	}
	if got := config.DateKey(atMidnight); got != time.Date(2026, 8, 18, 0, 0, 0, 0, time.UTC) {
		t.Fatalf("date key at business midnight = %s", got)
	}
}

func TestDayRangeUsesBusinessMidnight(t *testing.T) {
	key, err := config.ParseDate("2026-08-17")
	if err != nil {
		t.Fatalf("parse date: %v", err)
	}
	start, end := config.DayRange(key)
	if !start.Equal(time.Date(2026, 8, 16, 16, 0, 0, 0, time.UTC)) {
		t.Fatalf("day start = %s", start)
	}
	if !end.Equal(time.Date(2026, 8, 17, 16, 0, 0, 0, time.UTC)) {
		t.Fatalf("day end = %s", end)
	}
}
