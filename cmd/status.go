package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
	"github.com/user/jr/db"
	"github.com/user/jr/systemd"
)

var statusJSON bool

var statusCmd = &cobra.Command{
	Use:   "status <id|unit>",
	Short: "Show detailed status for a job",
	Args:  cobra.ExactArgs(1),
	RunE:  runStatus,
}

func init() {
	statusCmd.Flags().BoolVar(&statusJSON, "json", false, "output as JSON")
}

func runStatus(cmd *cobra.Command, args []string) error {
	job, err := db.FindJobByPartial(args[0])
	if err != nil {
		return fmt.Errorf("failed to find job: %w", err)
	}
	if job == nil {
		return fmt.Errorf("job not found: %s", args[0])
	}

	info, err := systemd.ShowUnit(job.Unit)
	if err != nil {
		info = &systemd.UnitInfo{Unit: job.Unit}
	}

	if statusJSON {
		return outputStatusJSON(job, info)
	}

	return outputStatusHuman(job, info)
}

func outputStatusHuman(job *db.Job, info *systemd.UnitInfo) error {
	fmt.Printf("Job:         %d\n", job.ID)
	fmt.Printf("Name:        %s\n", job.Name)
	fmt.Printf("Unit:        %s\n", job.Unit)

	created, _ := time.Parse(time.RFC3339, job.CreatedAtUTC)
	fmt.Printf("Created:     %s\n", created.Format(time.RFC3339))

	fmt.Printf("State:       %s\n", systemd.GetStateString(info))
	if info.SubState != "" {
		fmt.Printf("SubState:    %s\n", info.SubState)
	}

	if info.ExecMainPID != "" && info.ExecMainPID != "0" {
		fmt.Printf("PID:         %s\n", info.ExecMainPID)
	}

	if info.ExecMainStatus != "" {
		fmt.Printf("Exit Code:   %s\n", info.ExecMainStatus)
	}

	if info.ExecMainStartTimestamp != "" {
		fmt.Printf("Started:     %s\n", info.ExecMainStartTimestamp)
	}

	if info.ExecMainExitTimestamp != "" {
		fmt.Printf("Exited:      %s\n", info.ExecMainExitTimestamp)
	}

	fmt.Printf("Working Dir: %s\n", job.Cwd)

	var argv []string
	if job.ArgvJSON != "" {
		if err := json.Unmarshal([]byte(job.ArgvJSON), &argv); err != nil {
			fmt.Printf("Command:     <unmarshal error>\n")
		} else {
			fmt.Printf("Command:     %s\n", formatArgv(argv))
		}
	}

	if job.Host.Valid {
		fmt.Printf("Host:        %s\n", job.Host.String)
	}
	if job.User.Valid {
		fmt.Printf("User:        %s\n", job.User.String)
	}

	return nil
}

func outputStatusJSON(job *db.Job, info *systemd.UnitInfo) error {
	output := map[string]interface{}{
		"id":          job.ID,
		"name":        job.Name,
		"unit":        job.Unit,
		"created":     job.CreatedAtUTC,
		"state":       systemd.GetStateString(info),
		"activeState": info.ActiveState,
		"subState":    info.SubState,
		"pid":         info.ExecMainPID,
		"exitCode":    info.ExecMainStatus,
		"cwd":         job.Cwd,
	}

	var argv []string
	if job.ArgvJSON != "" {
		if err := json.Unmarshal([]byte(job.ArgvJSON), &argv); err == nil {
			output["argv"] = argv
		}
	}

	var env map[string]string
	if job.EnvJSON != "" {
		if err := json.Unmarshal([]byte(job.EnvJSON), &env); err == nil {
			output["env"] = env
		}
	}

	if info.ExecMainStartTimestamp != "" {
		output["started"] = info.ExecMainStartTimestamp
	}
	if info.ExecMainExitTimestamp != "" {
		output["exited"] = info.ExecMainExitTimestamp
	}
	if job.Host.Valid {
		output["host"] = job.Host.String
	}
	if job.User.Valid {
		output["user"] = job.User.String
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(output)
}

func formatArgv(argv []string) string {
	result := ""
	for i, arg := range argv {
		if i > 0 {
			result += " "
		}
		if containsSpace(arg) {
			result += fmt.Sprintf("%q", arg)
		} else {
			result += arg
		}
	}
	return result
}

func containsSpace(s string) bool {
	for _, r := range s {
		if r == ' ' || r == '\t' || r == '\n' {
			return true
		}
	}
	return false
}
