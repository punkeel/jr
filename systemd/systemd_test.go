package systemd

import (
	"strings"
	"testing"
)

func TestSanitizeName(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"hello", "hello"},
		{"Hello World", "hello_world"},
		{"test-123.txt", "test-123.txt"},
		{"UPPER_CASE", "upper_case"},
		{"special!@#$%chars", "special_____chars"},
		{"unicode:日本語", "unicode____"},
		{"", ""},
		{"a.b-c_d", "a.b-c_d"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := sanitizeName(tt.input)
			if result != tt.expected {
				t.Errorf("sanitizeName(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestGenerateUnitName(t *testing.T) {
	name := "test-job"
	unit := GenerateUnitName(name)

	// Check prefix
	if !strings.HasPrefix(unit, "jr-") {
		t.Errorf("GenerateUnitName(%q) = %q, expected prefix 'jr-'", name, unit)
	}

	// Check suffix
	if !strings.HasSuffix(unit, ".service") {
		t.Errorf("GenerateUnitName(%q) = %q, expected suffix '.service'", name, unit)
	}

	// Check that sanitized name is included
	if !strings.Contains(unit, "test-job") {
		t.Errorf("GenerateUnitName(%q) = %q, expected to contain 'test-job'", name, unit)
	}

	// Check format includes timestamp and ULID
	parts := strings.Split(strings.TrimSuffix(unit, ".service"), "-")
	if len(parts) < 4 {
		t.Errorf("GenerateUnitName(%q) = %q, expected format jr-name-timestamp-ulid.service", name, unit)
	}
}

func TestParseShowOutput(t *testing.T) {
	output := `ActiveState=active
SubState=running
ExecMainStatus=0
ExecMainPID=12345
ExecMainStartTimestamp=Mon 2024-01-01 12:00:00 UTC
ExecMainExitTimestamp=

ActiveState=inactive
SubState=dead
ExecMainStatus=1
ExecMainPID=0
ExecMainStartTimestamp=
ExecMainExitTimestamp=Mon 2024-01-01 13:00:00 UTC
`

	units := []string{"test1.service", "test2.service"}
	result := parseShowOutput(output, units)

	if len(result) != 2 {
		t.Fatalf("Expected 2 units, got %d", len(result))
	}

	// Check first unit
	info1 := result["test1.service"]
	if info1 == nil {
		t.Fatal("Expected info for test1.service")
	}
	if info1.ActiveState != "active" {
		t.Errorf("Expected ActiveState=active, got %q", info1.ActiveState)
	}
	if info1.SubState != "running" {
		t.Errorf("Expected SubState=running, got %q", info1.SubState)
	}
	if info1.ExecMainStatus != "0" {
		t.Errorf("Expected ExecMainStatus=0, got %q", info1.ExecMainStatus)
	}
	if info1.ExecMainPID != "12345" {
		t.Errorf("Expected ExecMainPID=12345, got %q", info1.ExecMainPID)
	}

	// Check second unit
	info2 := result["test2.service"]
	if info2 == nil {
		t.Fatal("Expected info for test2.service")
	}
	if info2.ActiveState != "inactive" {
		t.Errorf("Expected ActiveState=inactive, got %q", info2.ActiveState)
	}
	if info2.SubState != "dead" {
		t.Errorf("Expected SubState=dead, got %q", info2.SubState)
	}
	if info2.ExecMainStatus != "1" {
		t.Errorf("Expected ExecMainStatus=1, got %q", info2.ExecMainStatus)
	}
}

func TestParseShowOutputEmpty(t *testing.T) {
	result := parseShowOutput("", []string{"test.service"})

	if len(result) != 1 {
		t.Fatalf("Expected 1 unit, got %d", len(result))
	}

	info := result["test.service"]
	if info == nil {
		t.Fatal("Expected info for test.service")
	}
	if info.Unit != "test.service" {
		t.Errorf("Expected Unit=test.service, got %q", info.Unit)
	}
}

func TestGetStateString(t *testing.T) {
	tests := []struct {
		name     string
		info     *UnitInfo
		expected string
	}{
		{
			name:     "active",
			info:     &UnitInfo{ActiveState: "active"},
			expected: "active",
		},
		{
			name:     "inactive success",
			info:     &UnitInfo{ActiveState: "inactive", ExecMainStatus: "0"},
			expected: "exited",
		},
		{
			name:     "inactive failure",
			info:     &UnitInfo{ActiveState: "inactive", ExecMainStatus: "1"},
			expected: "failed",
		},
		{
			name:     "failed",
			info:     &UnitInfo{ActiveState: "failed"},
			expected: "failed",
		},
		{
			name:     "activating",
			info:     &UnitInfo{ActiveState: "activating"},
			expected: "activating",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetStateString(tt.info)
			if result != tt.expected {
				t.Errorf("GetStateString() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestShortenCommand(t *testing.T) {
	tests := []struct {
		name     string
		argv     []string
		maxLen   int
		expected string
	}{
		{
			name:     "empty",
			argv:     []string{},
			maxLen:   30,
			expected: "",
		},
		{
			name:     "single arg",
			argv:     []string{"python"},
			maxLen:   30,
			expected: "python",
		},
		{
			name:     "fits",
			argv:     []string{"python", "train.py"},
			maxLen:   30,
			expected: "python train.py",
		},
		{
			name:     "truncated",
			argv:     []string{"python", "train.py", "--epochs", "100", "--batch-size", "32"},
			maxLen:   25,
			expected: "python train.py --epochs ...",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ShortenCommand(tt.argv, tt.maxLen)
			if result != tt.expected {
				t.Errorf("ShortenCommand(%v, %d) = %q, want %q", tt.argv, tt.maxLen, result, tt.expected)
			}
		})
	}
}

func TestCommandExists(t *testing.T) {
	// Test with known commands
	if !CommandExists("sh") {
		t.Error("Expected CommandExists('sh') to be true")
	}

	if !CommandExists("/bin/sh") {
		t.Error("Expected CommandExists('/bin/sh') to be true")
	}

	// Test with non-existent command
	if CommandExists("definitely-not-a-real-command-12345") {
		t.Error("Expected CommandExists to be false for non-existent command")
	}
}
