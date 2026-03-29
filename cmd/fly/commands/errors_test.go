package commands

import (
	"errors"
	"testing"
)

func TestGetExitCode(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		wantCode int
	}{
		{"nil", nil, ExitCodeSuccess},
		{"CLIError auth", NewCLIError(errors.New("x"), ExitCodeAuthError, "auth"), ExitCodeAuthError},
		{"CLIError config", NewCLIError(errors.New("x"), ExitCodeConfigError, "config"), ExitCodeConfigError},
		{"wrapped CLIError", func() error {
			return NewCLIError(errors.New("inner"), ExitCodeValidationError, "invalid")
		}(), ExitCodeValidationError},
		{"network", errors.New("connection refused"), ExitCodeNetworkError},
		{"timeout", errors.New("dial timeout"), ExitCodeNetworkError},
		{"unauthorized", errors.New("unauthorized"), ExitCodeAuthError},
		{"token", errors.New("invalid token"), ExitCodeAuthError},
		{"validation", errors.New("invalid input"), ExitCodeValidationError},
		{"required", errors.New("field required"), ExitCodeValidationError},
		{"config", errors.New("config not found"), ExitCodeConfigError},
		{"generic", errors.New("something broke"), ExitCodeGeneralError},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetExitCode(tt.err)
			if got != tt.wantCode {
				t.Errorf("GetExitCode() = %d, want %d", got, tt.wantCode)
			}
		})
	}
}

func TestCLIError_Error(t *testing.T) {
	err := errors.New("underlying")
	ce := NewCLIError(err, ExitCodeGeneralError, "user message")
	if ce.Error() != "user message" {
		t.Errorf("Error() = %q, want %q", ce.Error(), "user message")
	}
	ce2 := NewCLIError(err, ExitCodeGeneralError, "")
	if ce2.Error() != "underlying" {
		t.Errorf("Error() with empty message = %q, want %q", ce2.Error(), "underlying")
	}
}

func TestCLIError_Unwrap(t *testing.T) {
	inner := errors.New("inner")
	ce := NewCLIError(inner, ExitCodeGeneralError, "msg")
	if !errors.Is(ce, inner) {
		t.Error("Unwrap should allow errors.Is to find inner")
	}
}
