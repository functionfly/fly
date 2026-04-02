package version

import (
	"strings"
	"testing"
	"time"
)

func TestBuildDate_Empty(t *testing.T) {
	saved := Date
	Date = ""
	defer func() { Date = saved }()

	got := BuildDate()
	if !got.IsZero() {
		t.Errorf("BuildDate() with empty Date should be zero time, got %v", got)
	}
}

func TestBuildDate_Invalid(t *testing.T) {
	saved := Date
	Date = "not-a-date"
	defer func() { Date = saved }()

	got := BuildDate()
	if !got.IsZero() {
		t.Errorf("BuildDate() with invalid Date should be zero time, got %v", got)
	}
}

func TestBuildDate_Valid(t *testing.T) {
	saved := Date
	Date = "2026-01-15T10:30:00Z"
	defer func() { Date = saved }()

	got := BuildDate()
	if got.IsZero() {
		t.Error("BuildDate() should not be zero time for valid RFC3339 date")
	}
	if got.Year() != 2026 {
		t.Errorf("Year = %d, want 2026", got.Year())
	}
	if got.Month() != time.January {
		t.Errorf("Month = %v, want January", got.Month())
	}
	if got.Day() != 15 {
		t.Errorf("Day = %d, want 15", got.Day())
	}
}

func TestInfo(t *testing.T) {
	sv, sc, sd := Version, Commit, Date
	Version = "1.2.3"
	Commit = "abc123"
	Date = "2026-01-01T00:00:00Z"
	defer func() { Version, Commit, Date = sv, sc, sd }()

	info := Info()
	if !strings.Contains(info, "1.2.3") {
		t.Errorf("Info() should contain version, got: %s", info)
	}
	if !strings.Contains(info, "abc123") {
		t.Errorf("Info() should contain commit, got: %s", info)
	}
	if !strings.Contains(info, "2026-01-01") {
		t.Errorf("Info() should contain date, got: %s", info)
	}
}

func TestShort_Dev(t *testing.T) {
	saved := Version
	Version = "dev"
	defer func() { Version = saved }()

	got := Short()
	if got != "dev" {
		t.Errorf("Short() = %q, want dev", got)
	}
}

func TestShort_Release(t *testing.T) {
	saved := Version
	Version = "2.0.0"
	defer func() { Version = saved }()

	got := Short()
	if got != "2.0.0" {
		t.Errorf("Short() = %q, want 2.0.0", got)
	}
}

func TestVersion_Default(t *testing.T) {
	if Version == "" {
		t.Error("Version should not be empty by default")
	}
}
