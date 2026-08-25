package cli

import (
	"strings"
	"testing"
)

func TestRun_NoArgs(t *testing.T) {
	stdout, _, code := captureOutput(t, func() int { return Run(nil) })
	if code != 1 {
		t.Errorf("code = %d, want 1", code)
	}
	if !strings.Contains(stdout, "Usage:") {
		t.Errorf("stdout = %q, want usage text", stdout)
	}
}

func TestRun_HelpVariants(t *testing.T) {
	for _, arg := range []string{"-h", "--help", "help"} {
		t.Run(arg, func(t *testing.T) {
			stdout, _, code := captureOutput(t, func() int { return Run([]string{arg}) })
			if code != 0 {
				t.Errorf("code = %d, want 0", code)
			}
			if !strings.Contains(stdout, "Usage:") {
				t.Errorf("stdout = %q, want usage text", stdout)
			}
		})
	}
}

func TestRun_UnknownCommand(t *testing.T) {
	stdout, stderr, code := captureOutput(t, func() int { return Run([]string{"bogus"}) })
	if code != 1 {
		t.Errorf("code = %d, want 1", code)
	}
	if !strings.Contains(stderr, "unknown command: bogus") {
		t.Errorf("stderr = %q, want it to mention the unknown command", stderr)
	}
	if !strings.Contains(stdout, "Usage:") {
		t.Errorf("stdout = %q, want usage text", stdout)
	}
}

// TestRun_DispatchRequiresConfig exercises Run's dispatch table for every
// command that loads config first: with no config initialized, each should
// reach loadConfig, fail the same way, and propagate that failure back
// through its own errf/return-1 path.
func TestRun_DispatchRequiresConfig(t *testing.T) {
	cases := []struct {
		cmd  string
		args []string
	}{
		{"add", []string{"~/.gitconfig"}},
		{"remove", []string{"~/.gitconfig"}},
		{"rm", []string{"~/.gitconfig"}},
		{"install", nil},
		{"apply", nil},
		{"status", nil},
		{"scan", nil},
		{"push", nil},
		{"pull", nil},
	}
	for _, tt := range cases {
		t.Run(tt.cmd, func(t *testing.T) {
			newCLIEnv(t)
			args := append([]string{tt.cmd}, tt.args...)
			_, stderr, code := captureOutput(t, func() int { return Run(args) })
			if code != 1 {
				t.Errorf("code = %d, want 1", code)
			}
			if !strings.Contains(stderr, "not initialized yet") {
				t.Errorf("stderr = %q, want it to mention dotmgr isn't initialized", stderr)
			}
		})
	}
}
