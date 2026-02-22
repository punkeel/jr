package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/user/jr/db"
	"github.com/user/jr/systemd"
)

var (
	rmStop      bool
	rmPurgeUnit bool
)

var rmCmd = &cobra.Command{
	Use:   "rm <id|unit>",
	Short: "Remove a job from the registry",
	Args:  cobra.ExactArgs(1),
	RunE:  runRm,
}

func init() {
	rmCmd.Flags().BoolVar(&rmStop, "stop", false, "stop the job before removing")
	rmCmd.Flags().BoolVar(&rmPurgeUnit, "purge-unit", false, "reset-failed after stopping")
}

func runRm(cmd *cobra.Command, args []string) error {
	job, err := db.FindJobByPartial(args[0])
	if err != nil {
		return fmt.Errorf("failed to find job: %w", err)
	}
	if job == nil {
		return fmt.Errorf("job not found: %s", args[0])
	}

	if rmStop {
		if err := systemd.StopUnit(job.Unit); err != nil {
			fmt.Printf("Warning: failed to stop unit: %v\n", err)
		}

		if rmPurgeUnit {
			if err := systemd.ResetFailedUnit(job.Unit); err != nil {
				fmt.Printf("Warning: failed to reset-failed: %v\n", err)
			}
		}
	}

	if err := db.DeleteJob(job.ID); err != nil {
		return fmt.Errorf("failed to delete job: %w", err)
	}

	fmt.Printf("Removed %d %s\n", job.ID, job.Unit)
	return nil
}
