package gtime

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var dateUnitPattern = regexp.MustCompile(`^(\d+)([dwMy])$`)

// ParseInterval parses an interval with support for all units that Grafana uses.
// An interval is relative to the current wall time.
func ParseInterval(inp string) (time.Duration, error) {
	dur, period, err := parse(inp)
	if err != nil {
		return 0, err
	}
	if period == "" {
		return dur, nil
	}

	num := int(dur)

	// Use UTC to ensure that the interval is deterministic, and daylight saving
	// doesn't cause surprises
	now := time.Now().UTC()
	switch period {
	case "d":
		return now.AddDate(0, 0, num).Sub(now), nil
	case "w":
		return now.AddDate(0, 0, num*7).Sub(now), nil
	case "M":
		return now.AddDate(0, num, 0).Sub(now), nil
	case "y":
		return now.AddDate(num, 0, 0).Sub(now), nil
	}

	return 0, fmt.Errorf("invalid interval %q", inp)
}

// ParseDuration parses a duration with support for all units that Grafana uses.
// Durations are independent of wall time.
func ParseDuration(inp string) (time.Duration, error) {
	dur, period, err := parse(inp)
	if err != nil {
		return 0, err
	}
	if period == "" {
		return dur, nil
	}

	// The average number of days in a year, using the Julian calendar
	const daysInAYear = 365.25
	const day = 24 * time.Hour
	const week = 7 * day
	const year = time.Duration(float64(day) * daysInAYear)
	const month = time.Duration(float64(year) / 12)

	switch period {
	case "d":
		return dur * day, nil
	case "w":
		return dur * week, nil
	case "M":
		return dur * month, nil
	case "y":
		return dur * year, nil
	}

	return 0, fmt.Errorf("invalid duration %q", inp)
}

func parse(inp string) (time.Duration, string, error) {
	result := dateUnitPattern.FindSubmatch([]byte(inp))
	if len(result) != 3 {
		dur, err := time.ParseDuration(inp)
		return dur, "", err
	}

	num, err := strconv.Atoi(string(result[1]))
	if err != nil {
		return 0, "", err
	}

	return time.Duration(num), string(result[2]), nil
}

// FormatInterval converts a duration into the units that Grafana uses
func FormatInterval(inter time.Duration) string {
	year := time.Hour * 24 * 365
	day := time.Hour * 24

	if inter >= year {
		return fmt.Sprintf("%dy", inter/year)
	}

	if inter >= day {
		return fmt.Sprintf("%dd", inter/day)
	}

	if inter >= time.Hour {
		return fmt.Sprintf("%dh", inter/time.Hour)
	}

	if inter >= time.Minute {
		return fmt.Sprintf("%dm", inter/time.Minute)
	}

	if inter >= time.Second {
		return fmt.Sprintf("%ds", inter/time.Second)
	}

	if inter >= time.Millisecond {
		return fmt.Sprintf("%dms", inter/time.Millisecond)
	}

	return "1ms"
}

// GetIntervalFrom returns the minimum interval.
// dsInterval is the string representation of data source min interval, if configured.
// queryInterval is the string representation of query interval (min interval), e.g. "10ms" or "10s".
// queryIntervalMS is a pre-calculated numeric representation of the query interval in milliseconds.
func GetIntervalFrom(dsInterval, queryInterval string, queryIntervalMS int64, defaultInterval time.Duration) (time.Duration, error) {
	// Apparently we are setting default value of queryInterval to 0s now
	interval := queryInterval
	if interval == "0s" {
		interval = ""
	}
	if interval == "" {
		if queryIntervalMS != 0 {
			return time.Duration(queryIntervalMS) * time.Millisecond, nil
		}
	}
	if interval == "" && dsInterval != "" {
		interval = dsInterval
	}
	if interval == "" {
		return defaultInterval, nil
	}

	parsedInterval, err := ParseIntervalStringToTimeDuration(interval)
	if err != nil {
		return time.Duration(0), err
	}

	return parsedInterval, nil
}

// ParseIntervalStringToTimeDuration converts a string representation of a expected (i.e. 1m30s) to time.Duration
// this method copied from grafana/grafana/pkg/tsdb/intervalv2.go
func ParseIntervalStringToTimeDuration(interval string) (time.Duration, error) {
	formattedInterval := strings.Replace(strings.Replace(interval, "<", "", 1), ">", "", 1)
	isPureNum, err := regexp.MatchString(`^\d+$`, formattedInterval)
	if err != nil {
		return time.Duration(0), err
	}
	if isPureNum {
		formattedInterval += "s"
	}
	parsedInterval, err := ParseDuration(formattedInterval)
	if err != nil {
		return time.Duration(0), err
	}
	return parsedInterval, nil
}
