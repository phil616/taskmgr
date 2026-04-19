package timeutil

import (
	"fmt"
	"os"
	"time"
)

const TimezoneName = "Asia/Shanghai"

var location = mustLoadLocation()

func mustLoadLocation() *time.Location {
	loc, err := time.LoadLocation(TimezoneName)
	if err != nil {
		panic(fmt.Sprintf("failed to load timezone %s: %v", TimezoneName, err))
	}
	return loc
}

func Location() *time.Location {
	return location
}

func SetLocalTimezone() error {
	if err := os.Setenv("TZ", TimezoneName); err != nil {
		return err
	}
	time.Local = location
	return nil
}

func Now() time.Time {
	return time.Now().In(location)
}

func Normalize(t time.Time) time.Time {
	if t.IsZero() {
		return t
	}
	return t.In(location)
}

func NormalizePtr(t *time.Time) *time.Time {
	if t == nil {
		return nil
	}
	normalized := Normalize(*t)
	return &normalized
}

func StartOfDay(t time.Time) time.Time {
	tt := t.In(location)
	return time.Date(tt.Year(), tt.Month(), tt.Day(), 0, 0, 0, 0, location)
}

func ParseDate(value string) (time.Time, error) {
	return time.ParseInLocation("2006-01-02", value, location)
}

func ParseDateTime(value string) (time.Time, error) {
	formats := []string{
		time.RFC3339,
		"2006-01-02",
		"2006-01-02T15:04:05",
		"2006-01-02T15:04",
		"2006-01-02 15:04:05",
		"2006-01-02 15:04",
	}
	for _, format := range formats {
		if format == time.RFC3339 {
			t, err := time.Parse(format, value)
			if err == nil {
				return t.In(location), nil
			}
			continue
		}
		t, err := time.ParseInLocation(format, value, location)
		if err == nil {
			return t.In(location), nil
		}
	}
	return time.Time{}, fmt.Errorf("无法解析时间: %s", value)
}
