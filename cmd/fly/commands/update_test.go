package commands

import (
	"testing"
)

func TestBumpVersion_Patch(t *testing.T) {
	tests := []struct {
		current string
		want    string
	}{
		{"1.0.0", "1.0.1"},
		{"1.0.99", "1.0.100"},
		{"0.0.0", "0.0.1"},
		{"10.20.30", "10.20.31"},
	}
	for _, tt := range tests {
		got, err := bumpVersion(tt.current, "patch")
		if err != nil {
			t.Errorf("bumpVersion(%q, patch) unexpected error: %v", tt.current, err)
			continue
		}
		if got != tt.want {
			t.Errorf("bumpVersion(%q, patch) = %q, want %q", tt.current, got, tt.want)
		}
	}
}

func TestBumpVersion_Minor(t *testing.T) {
	tests := []struct {
		current string
		want    string
	}{
		{"1.0.0", "1.1.0"},
		{"1.5.99", "1.6.0"},
		{"0.0.5", "0.1.0"},
	}
	for _, tt := range tests {
		got, err := bumpVersion(tt.current, "minor")
		if err != nil {
			t.Errorf("bumpVersion(%q, minor) unexpected error: %v", tt.current, err)
			continue
		}
		if got != tt.want {
			t.Errorf("bumpVersion(%q, minor) = %q, want %q", tt.current, got, tt.want)
		}
	}
}

func TestBumpVersion_Major(t *testing.T) {
	tests := []struct {
		current string
		want    string
	}{
		{"1.0.0", "2.0.0"},
		{"0.9.99", "1.0.0"},
		{"3.14.15", "4.0.0"},
	}
	for _, tt := range tests {
		got, err := bumpVersion(tt.current, "major")
		if err != nil {
			t.Errorf("bumpVersion(%q, major) unexpected error: %v", tt.current, err)
			continue
		}
		if got != tt.want {
			t.Errorf("bumpVersion(%q, major) = %q, want %q", tt.current, got, tt.want)
		}
	}
}

func TestBumpVersion_ExplicitVersion(t *testing.T) {
	tests := []struct {
		current string
		bump    string
		want    string
	}{
		{"1.0.0", "2.0.0", "2.0.0"},
		{"1.0.0", "0.1.0", "0.1.0"},
		{"1.0.0", "10.20.30", "10.20.30"},
		{"1.0.0", "v2.0.0", "2.0.0"}, // v prefix stripped
	}
	for _, tt := range tests {
		got, err := bumpVersion(tt.current, tt.bump)
		if err != nil {
			t.Errorf("bumpVersion(%q, %q) unexpected error: %v", tt.current, tt.bump, err)
			continue
		}
		if got != tt.want {
			t.Errorf("bumpVersion(%q, %q) = %q, want %q", tt.current, tt.bump, got, tt.want)
		}
	}
}

func TestBumpVersion_VPrefixOnCurrent(t *testing.T) {
	got, err := bumpVersion("v1.2.3", "patch")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "1.2.4" {
		t.Errorf("bumpVersion(v1.2.3, patch) = %q, want %q", got, "1.2.4")
	}
}

func TestBumpVersion_InvalidVersion(t *testing.T) {
	invalid := []string{"", "1.0", "1", "abc", "1.0.0.0", "1.0.x"}
	for _, v := range invalid {
		_, err := bumpVersion(v, "patch")
		if err == nil {
			t.Errorf("bumpVersion(%q, patch) should return error", v)
		}
	}
}

func TestBumpVersion_InvalidBump(t *testing.T) {
	invalid := []string{"", "invalid", "1.0", "abc"}
	for _, b := range invalid {
		_, err := bumpVersion("1.0.0", b)
		if err == nil {
			t.Errorf("bumpVersion(1.0.0, %q) should return error", b)
		}
	}
}
