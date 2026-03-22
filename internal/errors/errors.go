package errors

import (
	"fmt"
	"strings"
)

// Exit codes
const (
	ExitOK          = 0
	ExitUserError   = 1 // fixable by the user (bad args, not configured, etc.)
	ExitSystemError = 2 // network, gateway down, etc.
)

// Error is a structured error with context and hints for LLM agents.
type Error struct {
	Code      Code
	Message   string
	Hint      string
	Available []string // list of valid options when something isn't found
	Cause     error
}

func (e *Error) Error() string {
	var b strings.Builder
	b.WriteString("error: ")
	b.WriteString(e.Message)

	if len(e.Available) > 0 {
		b.WriteString("\nAvailable:")
		for _, a := range e.Available {
			b.WriteString("\n  ")
			b.WriteString(a)
		}
	}

	if e.Hint != "" {
		b.WriteString("\nHint: ")
		b.WriteString(e.Hint)
	}

	return b.String()
}

func (e *Error) Unwrap() error {
	return e.Cause
}

// ExitCode returns the appropriate exit code for this error.
func (e *Error) ExitCode() int {
	switch e.Code {
	case CodeNotFound, CodeInvalidArg, CodeNotConfigured, CodeAuthExpired:
		return ExitUserError
	default:
		return ExitSystemError
	}
}

// Code identifies the category of error.
type Code int

const (
	CodeNotFound       Code = iota // device/speaker/service not found
	CodeUnreachable                // device exists but is offline
	CodeNotConfigured              // provider not set up
	CodeAuthExpired                // token expired, needs re-auth
	CodeNoSession                  // no active playback session
	CodeInvalidArg                 // bad user input
	CodeNetwork                    // network/connection failure
	CodeProviderError              // provider-specific error
)

// NotFound creates a "not found" error with available alternatives.
func NotFound(kind, query string, available []string) *Error {
	return &Error{
		Code:      CodeNotFound,
		Message:   fmt.Sprintf("no %s matching '%s'", kind, query),
		Available: available,
		Hint:      fmt.Sprintf("use 'crib %ss list' to see all %ss", kind, kind),
	}
}

// Unreachable creates an error for offline devices.
func Unreachable(name string) *Error {
	return &Error{
		Code:    CodeUnreachable,
		Message: fmt.Sprintf("'%s' is unreachable (powered off at the switch)", name),
		Hint:    "nothing you can do remotely — the device needs to be physically powered on",
	}
}

// NotConfigured creates an error for missing provider setup.
func NotConfigured(provider string) *Error {
	return &Error{
		Code:    CodeNotConfigured,
		Message: fmt.Sprintf("%s is not configured", provider),
		Hint:    "run 'crib setup' to configure it",
	}
}

// AuthExpired creates an error for expired tokens.
func AuthExpired(provider string) *Error {
	return &Error{
		Code:    CodeAuthExpired,
		Message: fmt.Sprintf("%s session expired", provider),
		Hint:    fmt.Sprintf("run 'crib %s login' to re-authenticate", provider),
	}
}

// NoSession creates an error for missing playback sessions.
func NoSession(provider string) *Error {
	return &Error{
		Code:    CodeNoSession,
		Message: fmt.Sprintf("no active %s session", provider),
		Hint:    fmt.Sprintf("open %s on a device first, or use 'crib speakers play <room> <query>' to play on a speaker directly", provider),
	}
}

// InvalidArg creates an error for bad user input.
func InvalidArg(message string) *Error {
	return &Error{
		Code:    CodeInvalidArg,
		Message: message,
	}
}

// InvalidArgWithHint creates an error for bad input with a usage hint.
func InvalidArgWithHint(message, hint string) *Error {
	return &Error{
		Code:    CodeInvalidArg,
		Message: message,
		Hint:    hint,
	}
}

// Network creates an error for connectivity issues.
func Network(provider, host string, cause error) *Error {
	return &Error{
		Code:    CodeNetwork,
		Message: fmt.Sprintf("cannot reach %s at %s", provider, host),
		Hint:    "check that the device is powered on and on the same network.\nOn macOS, ensure Local Network access is enabled in System Settings > Privacy & Security.",
		Cause:   cause,
	}
}

// Provider creates a provider-specific error.
func Provider(provider string, cause error) *Error {
	return &Error{
		Code:    CodeProviderError,
		Message: fmt.Sprintf("%s error: %v", provider, cause),
		Cause:   cause,
	}
}

// ProviderMismatch creates an error for cross-provider operations.
func ProviderMismatch(operation, name1, provider1, name2, provider2 string) *Error {
	return &Error{
		Code:    CodeProviderError,
		Message: fmt.Sprintf("cannot %s '%s' (%s) with '%s' (%s)", operation, name1, provider1, name2, provider2),
		Hint:    fmt.Sprintf("%s only works between devices from the same provider", operation),
	}
}
