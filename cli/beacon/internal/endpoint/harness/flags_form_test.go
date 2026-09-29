package harness

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/asymptote-labs/agent-beacon/cli/beacon/internal/testenv"
)

// flagsFormCommand is the hook command the installer writes today: settings travel as --log and
// --config flags instead of an inline `BEACON_ENDPOINT_MODE=1 ...` prefix, which neither Windows
// shell accepts. Status must recognize it, or `status` and `doctor` report a working install as
// missing and tell the user to reinstall hooks that are already there.
func flagsFormCommand(platform string) string {
	return `'/Users/u/.beacon/endpoint/hooks/beacon-hooks' --platform ` + platform +
		` --log '/Users/u/.beacon/endpoint/logs/runtime.jsonl' --config '/Users/u/.beacon/endpoint/config.json' pre-tool`
}

// A hook another tool installed beside Beacon's, taken from a real Cursor hooks.json.
const foreignCommand = `cmux hooks cursor agent-response || echo '{}'`

func writeTemp(t *testing.T, name, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func jsonString(s string) string {
	quoted, _ := json.Marshal(s)
	return string(quoted)
}

func TestStatusRecognizesFlagsFormHooks(t *testing.T) {
	type check func(t *testing.T, command string) TelemetryStatus

	cursor := func(t *testing.T, command string) TelemetryStatus {
		home := t.TempDir()
		testenv.SetHome(t, home)
		t.Setenv("PATH", "") // keep discovery from probing a real cursor binary
		body := `{"version":1,"hooks":{"preToolUse":[{"command":` + jsonString(command) + `}]}}`
		if err := os.MkdirAll(filepath.Join(home, ".cursor"), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(home, ".cursor", "hooks.json"), []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
		return DiscoverCursor().TelemetryStatus
	}
	antigravity := func(t *testing.T, command string) TelemetryStatus {
		body := `{"beacon-endpoint":{"PreToolUse":[{"hooks":[{"type":"command","command":` + jsonString(command) + `}]}]}}`
		status, _ := antigravityStatus(writeTemp(t, "hooks.json", body))
		return status
	}
	devinCLI := func(t *testing.T, command string) TelemetryStatus {
		body := `{"hooks":{"PreToolUse":[{"hooks":[{"type":"command","command":` + jsonString(command) + `}]}]}}`
		status, _ := devinStatus(writeTemp(t, "config.json", body), "devin", "devin-cli")
		return status
	}
	devinDesktop := func(t *testing.T, command string) TelemetryStatus {
		body := `{"hooks":{"pre_run_command":[{"command":` + jsonString(command) + `}]}}`
		status, _ := windsurfCascadeStatus(writeTemp(t, "hooks.json", body), "devin-desktop")
		return status
	}
	hermes := func(t *testing.T, command string) TelemetryStatus {
		body := "hooks:\n  pre_tool_call:\n    - command: " + jsonString(command) + "\n"
		status, _ := hermesStatus(writeTemp(t, "config.yaml", body))
		return status
	}

	for name, tc := range map[string]struct {
		check    check
		platform string
	}{
		"cursor":        {cursor, "cursor"},
		"antigravity":   {antigravity, "antigravity"},
		"devin cli":     {devinCLI, "devin-cli"},
		"devin desktop": {devinDesktop, "devin-desktop"},
		"hermes":        {hermes, "hermes"},
	} {
		t.Run(name, func(t *testing.T) {
			if got := tc.check(t, flagsFormCommand(tc.platform)); got != TelemetryEnabled {
				t.Errorf("flags-form Beacon hook: status = %q, want %q", got, TelemetryEnabled)
			}
			if got := tc.check(t, flagsFormCommand("some-other-runtime")); got == TelemetryEnabled {
				t.Errorf("Beacon hook for another platform: status = %q, want not enabled", got)
			}
			if got := tc.check(t, foreignCommand); got == TelemetryEnabled {
				t.Errorf("foreign hook: status = %q, want not enabled", got)
			}
		})
	}
}
