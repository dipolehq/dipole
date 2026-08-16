// time_condition_test.go tests time-window evaluation: day matching, midnight crossing, timezone errors, and per-day windows.
package conditions

import (
	"testing"
)

func TestTimeCondition_WithinWindow_Passes(t *testing.T) {
	checker := &TimeConditionChecker{}
	cfg := &TimeConditionConfig{
		Days:      []string{"Mon", "Tue", "Wed", "Thu", "Fri", "Sat", "Sun"},
		StartTime: "00:00",
		EndTime:   "23:59",
		Timezone:  "UTC",
	}

	passed, reason, err := checker.Check(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if !passed {
		t.Errorf("expected pass, got fail: %s", reason)
	}
}

func TestTimeCondition_NoMatchingDay_Fails(t *testing.T) {
	checker := &TimeConditionChecker{}
	cfg := &TimeConditionConfig{
		Days:      []string{},
		StartTime: "00:00",
		EndTime:   "23:59",
		Timezone:  "UTC",
	}

	passed, _, err := checker.Check(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if passed {
		t.Error("expected fail with empty day list")
	}
}

func TestTimeCondition_MidnightCrossing(t *testing.T) {
	checker := &TimeConditionChecker{}
	cfg := &TimeConditionConfig{
		Days:      []string{"Mon", "Tue", "Wed", "Thu", "Fri", "Sat", "Sun"},
		StartTime: "22:00",
		EndTime:   "06:00",
		Timezone:  "UTC",
	}

	_, _, err := checker.Check(cfg)
	if err != nil {
		t.Fatal(err)
	}
}

func TestTimeCondition_InvalidTimezone_Error(t *testing.T) {
	checker := &TimeConditionChecker{}
	cfg := &TimeConditionConfig{
		Days:      []string{"Mon"},
		StartTime: "09:00",
		EndTime:   "18:00",
		Timezone:  "Invalid/Zone",
	}

	_, _, err := checker.Check(cfg)
	if err == nil {
		t.Error("expected error for invalid timezone")
	}
}

func TestTimeCondition_PerDayWindows(t *testing.T) {
	checker := &TimeConditionChecker{}
	cfg := &TimeConditionConfig{
		Windows: []DayWindow{
			{Day: "Mon", StartTime: "00:00", EndTime: "23:59"},
			{Day: "Tue", StartTime: "00:00", EndTime: "23:59"},
			{Day: "Wed", StartTime: "00:00", EndTime: "23:59"},
			{Day: "Thu", StartTime: "00:00", EndTime: "23:59"},
			{Day: "Fri", StartTime: "00:00", EndTime: "23:59"},
			{Day: "Sat", StartTime: "00:00", EndTime: "23:59"},
			{Day: "Sun", StartTime: "00:00", EndTime: "23:59"},
		},
		Timezone: "Asia/Kolkata",
	}

	passed, reason, err := checker.Check(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if !passed {
		t.Errorf("expected pass with all-day windows, got: %s", reason)
	}
}
