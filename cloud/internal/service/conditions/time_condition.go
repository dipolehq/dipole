/*
time_condition.go evaluates time-based access restrictions.

Checks if the current time (in the configured timezone) falls within the
allowed time window for today's day of week. Each day can have its own
start/end time or be absent (meaning that day is not allowed). Supports
midnight crossing (e.g., 22:00-06:00). Returns false if outside.
*/
package conditions

import (
	"fmt"
	"strings"
	"time"
)

type TimeConditionChecker struct{}

type TimeConditionConfig struct {
	// Per-day windows: key is day abbreviation (Mon, Tue, etc.)
	// Each entry has start_time and end_time in HH:MM format
	Windows  []DayWindow `json:"windows"`
	Timezone string      `json:"timezone"`

	// Legacy single-window format (days + start_time + end_time)
	Days      []string `json:"days"`
	StartTime string   `json:"start_time"`
	EndTime   string   `json:"end_time"`
}

type DayWindow struct {
	Day       string `json:"day"`
	StartTime string `json:"start_time"`
	EndTime   string `json:"end_time"`
}

func (c *TimeConditionChecker) Check(cfg *TimeConditionConfig) (bool, string, error) {
	loc, err := time.LoadLocation(cfg.Timezone)
	if err != nil {
		return false, "", fmt.Errorf("invalid timezone %q: %w", cfg.Timezone, err)
	}

	now := time.Now().In(loc)
	dayAbbr := now.Weekday().String()[:3]
	nowMin := now.Hour()*60 + now.Minute()

	// Per-day windows (preferred format)
	if len(cfg.Windows) > 0 {
		for _, w := range cfg.Windows {
			if strings.EqualFold(w.Day, dayAbbr) {
				return checkTimeWindow(nowMin, w.StartTime, w.EndTime, dayAbbr, cfg.Timezone, now)
			}
		}
		return false, fmt.Sprintf("Time condition failed: %s is not in allowed days", dayAbbr), nil
	}

	// Legacy single-window format
	if !dayInList(dayAbbr, cfg.Days) {
		return false, fmt.Sprintf("Time condition failed: %s is not in allowed days %v", dayAbbr, cfg.Days), nil
	}
	return checkTimeWindow(nowMin, cfg.StartTime, cfg.EndTime, dayAbbr, cfg.Timezone, now)
}

func checkTimeWindow(nowMin int, startTime, endTime, day, tz string, now time.Time) (bool, string, error) {
	startMin, err := parseHHMM(startTime)
	if err != nil {
		return false, "", err
	}
	endMin, err := parseHHMM(endTime)
	if err != nil {
		return false, "", err
	}

	var inWindow bool
	if startMin <= endMin {
		inWindow = nowMin >= startMin && nowMin < endMin
	} else {
		inWindow = nowMin >= startMin || nowMin < endMin
	}

	if !inWindow {
		return false, fmt.Sprintf("Time condition failed: current time %02d:%02d %s is outside allowed window %s-%s on %s",
			now.Hour(), now.Minute(), tz, startTime, endTime, day), nil
	}

	return true, "", nil
}

func dayInList(day string, days []string) bool {
	for _, d := range days {
		if strings.EqualFold(d, day) {
			return true
		}
	}
	return false
}

func parseHHMM(s string) (int, error) {
	var h, m int
	_, err := fmt.Sscanf(s, "%d:%d", &h, &m)
	if err != nil {
		return 0, fmt.Errorf("invalid time format %q (expected HH:MM): %w", s, err)
	}
	return h*60 + m, nil
}
