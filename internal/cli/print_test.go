package cli

import (
	"strings"
	"testing"
)

func TestStep(t *testing.T) {
	stdout, _, _ := captureOutput(t, func() int { step("doing %s", "thing"); return 0 })
	if !strings.Contains(stdout, "doing thing") {
		t.Errorf("stdout = %q, want it to contain the formatted message", stdout)
	}
}

func TestInfo(t *testing.T) {
	stdout, _, _ := captureOutput(t, func() int { info("detail %d", 42); return 0 })
	if !strings.Contains(stdout, "detail 42") {
		t.Errorf("stdout = %q, want it to contain the formatted message", stdout)
	}
}

func TestErrf(t *testing.T) {
	_, stderr, _ := captureOutput(t, func() int { errf("boom %s", "here"); return 0 })
	if !strings.Contains(stderr, "Error: boom here") {
		t.Errorf("stderr = %q, want it to contain the formatted error", stderr)
	}
}
