package errors

import (
	"fmt"
	"strings"
	"testing"
)

func TestNotFound(t *testing.T) {
	err := NotFound("device", "kitchn", []string{"küche (light, tradfri)", "kugel (light, tradfri)"})

	if err.Code != CodeNotFound {
		t.Errorf("expected CodeNotFound, got %d", err.Code)
	}
	if !strings.Contains(err.Error(), "kitchn") {
		t.Errorf("error should contain query: %s", err.Error())
	}
	if !strings.Contains(err.Error(), "küche") {
		t.Errorf("error should list alternatives: %s", err.Error())
	}
	if !strings.Contains(err.Error(), "Hint:") {
		t.Errorf("error should have a hint: %s", err.Error())
	}
	if err.ExitCode() != ExitUserError {
		t.Errorf("expected ExitUserError, got %d", err.ExitCode())
	}
}

func TestUnreachable(t *testing.T) {
	err := Unreachable("Deckenlampe")

	if err.Code != CodeUnreachable {
		t.Errorf("expected CodeUnreachable, got %d", err.Code)
	}
	if !strings.Contains(err.Error(), "Deckenlampe") {
		t.Errorf("error should contain device name: %s", err.Error())
	}
	if !strings.Contains(err.Error(), "physically") {
		t.Errorf("error should explain the situation: %s", err.Error())
	}
}

func TestNotConfigured(t *testing.T) {
	err := NotConfigured("Spotify")

	if !strings.Contains(err.Error(), "not configured") {
		t.Errorf("error should say not configured: %s", err.Error())
	}
	if !strings.Contains(err.Error(), "crib setup") {
		t.Errorf("error should suggest setup: %s", err.Error())
	}
}

func TestAuthExpired(t *testing.T) {
	err := AuthExpired("spotify")

	if err.Code != CodeAuthExpired {
		t.Errorf("expected CodeAuthExpired, got %d", err.Code)
	}
	if !strings.Contains(err.Error(), "login") {
		t.Errorf("error should suggest login: %s", err.Error())
	}
}

func TestNoSession(t *testing.T) {
	err := NoSession("Spotify")

	if !strings.Contains(err.Error(), "no active") {
		t.Errorf("error should say no active session: %s", err.Error())
	}
	if !strings.Contains(err.Error(), "speakers play") {
		t.Errorf("error should suggest alternative: %s", err.Error())
	}
}

func TestInvalidArg(t *testing.T) {
	err := InvalidArgWithHint("brightness must be 0-100, got '150'", "usage: crib devices dim <name> <0-100>")

	if err.Code != CodeInvalidArg {
		t.Errorf("expected CodeInvalidArg, got %d", err.Code)
	}
	if err.ExitCode() != ExitUserError {
		t.Errorf("expected ExitUserError, got %d", err.ExitCode())
	}
}

func TestNetwork(t *testing.T) {
	cause := fmt.Errorf("connection refused")
	err := Network("TRÅDFRI gateway", "192.168.178.21", cause)

	if err.Code != CodeNetwork {
		t.Errorf("expected CodeNetwork, got %d", err.Code)
	}
	if err.ExitCode() != ExitSystemError {
		t.Errorf("expected ExitSystemError, got %d", err.ExitCode())
	}
	if err.Unwrap() != cause {
		t.Errorf("Unwrap should return cause")
	}
	if !strings.Contains(err.Error(), "192.168.178.21") {
		t.Errorf("error should contain host: %s", err.Error())
	}
}

func TestProviderMismatch(t *testing.T) {
	err := ProviderMismatch("group", "Küche", "sonos", "Bedroom", "homepod")

	if !strings.Contains(err.Error(), "sonos") {
		t.Errorf("error should mention providers: %s", err.Error())
	}
	if !strings.Contains(err.Error(), "same provider") {
		t.Errorf("error should explain the constraint: %s", err.Error())
	}
}

func TestErrorFormat(t *testing.T) {
	err := NotFound("device", "test", []string{"one", "two"})
	output := err.Error()

	// Should start with "error: "
	if !strings.HasPrefix(output, "error: ") {
		t.Errorf("should start with 'error: ': %s", output)
	}

	// Should have Available section
	if !strings.Contains(output, "Available:") {
		t.Errorf("should have Available section: %s", output)
	}

	// Should have Hint section
	if !strings.Contains(output, "Hint:") {
		t.Errorf("should have Hint section: %s", output)
	}
}
