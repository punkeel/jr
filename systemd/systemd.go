package systemd

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/oklog/ulid/v2"
)

const unitPrefix = "jr-"

type UnitInfo struct {
	Unit                   string
	ActiveState            string
	SubState               string
	ExecMainStatus         string
	ExecMainPID            string
	ExecMainStartTimestamp string
	ExecMainExitTimestamp  string
}

func GenerateUnitName(name string) string {
	cleanName := sanitizeName(name)
	timestamp := time.Now().UTC().Format("20060102-150405")
	shortID := ulid.Make().String()[:8]
	return fmt.Sprintf("%s%s-%s-%s.service", unitPrefix, cleanName, timestamp, shortID)
}

func sanitizeName(name string) string {
	var result strings.Builder
	for _, r := range name {
		switch {
		case r >= 'a' && r <= 'z':
			result.WriteRune(r)
		case r >= 'A' && r <= 'Z':
			result.WriteRune(r + ('a' - 'A'))
		case r >= '0' && r <= '9':
			result.WriteRune(r)
		case r == '.' || r == '-' || r == '_':
			result.WriteRune(r)
		default:
			result.WriteRune('_')
		}
	}
	return result.String()
}

func StartUnit(unit, cwd string, argv []string, env map[string]string, props map[string]string, desc string) error {
	args := []string{
		"--user",
		"--unit", unit,
		"--working-directory", cwd,
		"--collect",
	}

	if desc != "" {
		args = append(args, "-p", "Description="+desc)
	}

	for k, v := range env {
		args = append(args, "--setenv", fmt.Sprintf("%s=%s", k, v))
	}

	for k, v := range props {
		args = append(args, "-p", fmt.Sprintf("%s=%s", k, v))
	}

	args = append(args, "--")
	args = append(args, argv...)

	cmd := exec.Command("systemd-run", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

func StopUnit(unit string) error {
	cmd := exec.Command("systemctl", "--user", "stop", unit)
	return cmd.Run()
}

func KillUnit(unit, signal string) error {
	cmd := exec.Command("systemctl", "--user", "kill", "-s", signal, unit)
	return cmd.Run()
}

func ResetFailedUnit(unit string) error {
	cmd := exec.Command("systemctl", "--user", "reset-failed", unit)
	return cmd.Run()
}

func ShowUnits(units []string) (map[string]*UnitInfo, error) {
	if len(units) == 0 {
		return make(map[string]*UnitInfo), nil
	}

	args := append([]string{"--user", "show"}, units...)
	args = append(args, "-p", "ActiveState", "-p", "SubState", "-p", "ExecMainStatus",
		"-p", "ExecMainPID", "-p", "ExecMainStartTimestamp", "-p", "ExecMainExitTimestamp")

	cmd := exec.Command("systemctl", args...)
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	return parseShowOutput(string(output), units), nil
}

func ShowUnit(unit string) (*UnitInfo, error) {
	infos, err := ShowUnits([]string{unit})
	if err != nil {
		return nil, err
	}

	info, ok := infos[unit]
	if !ok {
		return nil, fmt.Errorf("unit %s not found", unit)
	}

	return info, nil
}

func parseShowOutput(output string, units []string) map[string]*UnitInfo {
	result := make(map[string]*UnitInfo)

	for _, unit := range units {
		result[unit] = &UnitInfo{Unit: unit}
	}

	scanner := bufio.NewScanner(strings.NewReader(output))

	for scanner.Scan() {
		line := scanner.Text()

		if strings.HasPrefix(line, "ExecMainStatus=") ||
			strings.HasPrefix(line, "ActiveState=") ||
			strings.HasPrefix(line, "SubState=") ||
			strings.HasPrefix(line, "ExecMainPID=") ||
			strings.HasPrefix(line, "ExecMainStartTimestamp=") ||
			strings.HasPrefix(line, "ExecMainExitTimestamp=") {

			for _, unit := range units {
				info := result[unit]
				if info.ExecMainStatus == "" && strings.HasPrefix(line, "ExecMainStatus=") {
					info.ExecMainStatus = strings.TrimPrefix(line, "ExecMainStatus=")
					break
				}
				if info.ActiveState == "" && strings.HasPrefix(line, "ActiveState=") {
					info.ActiveState = strings.TrimPrefix(line, "ActiveState=")
					break
				}
				if info.SubState == "" && strings.HasPrefix(line, "SubState=") {
					info.SubState = strings.TrimPrefix(line, "SubState=")
					break
				}
			}
		}
	}

	return result
}

func Logs(unit string, follow bool, lines int, since, until string, noColor bool) error {
	args := []string{"--user", "-u", unit, "-o", "short-iso"}

	if follow {
		args = append(args, "-f")
	}

	if lines > 0 {
		args = append(args, "-n", fmt.Sprintf("%d", lines))
	}

	if since != "" {
		args = append(args, "--since", since)
	}

	if until != "" {
		args = append(args, "--until", until)
	}

	if noColor {
		args = append(args, "--no-pager")
	}

	cmd := exec.Command("journalctl", args...)

	if follow {
		cmd.Stdin = os.Stdin
	}
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

func CheckUserSystemd() error {
	cmd := exec.Command("systemctl", "--user", "status")
	return cmd.Run()
}

func CheckLingering() (bool, error) {
	user := os.Getenv("USER")
	if user == "" {
		return false, fmt.Errorf("USER environment variable not set")
	}

	cmd := exec.Command("loginctl", "show-user", user, "-p", "Linger")
	output, err := cmd.Output()
	if err != nil {
		return false, err
	}

	return strings.Contains(string(output), "yes"), nil
}

func CheckSystemdRun() error {
	_, err := exec.LookPath("systemd-run")
	return err
}

func CheckJournalctl() error {
	_, err := exec.LookPath("journalctl")
	return err
}

func GetStateString(info *UnitInfo) string {
	if info.ActiveState == "active" {
		return "active"
	} else if info.ActiveState == "inactive" {
		if info.ExecMainStatus != "" && info.ExecMainStatus != "0" {
			return "failed"
		}
		return "exited"
	} else if info.ActiveState == "failed" {
		return "failed"
	}
	return info.ActiveState
}

func CommandExists(cmd string) bool {
	if strings.Contains(cmd, "/") {
		_, err := os.Stat(cmd)
		return err == nil
	}

	_, err := exec.LookPath(cmd)
	return err == nil
}

func CommandExistsInPath(cmd string) bool {
	_, err := exec.LookPath(cmd)
	return err == nil
}

func ShortenCommand(argv []string, maxLen int) string {
	if len(argv) == 0 {
		return ""
	}

	cmd := filepath.Base(argv[0])
	for i := 1; i < len(argv); i++ {
		if len(cmd)+len(argv[i])+1 > maxLen {
			cmd += " ..."
			break
		}
		cmd += " " + argv[i]
	}

	return cmd
}
