package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/user/jr/db"
	"github.com/user/jr/systemd"
)

var stopSignal string

var stopCmd = &cobra.Command{
	Use:   "stop <id|unit>",
	Short: "Stop a running job",
	Args:  cobra.ExactArgs(1),
	RunE:  runStop,
}

func init() {
	stopCmd.Flags().StringVarP(&stopSignal, "signal", "s", "", "signal to send before stopping (e.g., SIGTERM, SIGINT)")
}

func runStop(cmd *cobra.Command, args []string) error {
	job, err := db.FindJobByPartial(args[0])
	if err != nil {
		return fmt.Errorf("failed to find job: %w", err)
	}
	if job == nil {
		return fmt.Errorf("job not found: %s", args[0])
	}

	if stopSignal != "" {
		if err := systemd.KillUnit(job.Unit, stopSignal); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to send signal: %v\n", err)
		}
	}

	if err := systemd.StopUnit(job.Unit); err != nil {
		return fmt.Errorf("failed to stop unit: %w", err)
	}

	if err := db.UpdateJobState(job.ID, "stopped"); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to update job state: %v\n", err)
	}

	fmt.Printf("Stopped %d %s\n", job.ID, job.Unit)
	return nil
}
