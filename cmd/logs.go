package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/user/jr/db"
	"github.com/user/jr/systemd"
)

var (
	logsFollow  bool
	logsLines   int
	logsSince   string
	logsUntil   string
	logsNoColor bool
)

var logsCmd = &cobra.Command{
	Use:     "logs <id|unit>",
	Short:   "Stream or print logs for a job",
	Aliases: []string{"tail", "attach"},
	Args:    cobra.ExactArgs(1),
	RunE:    runLogs,
}

func init() {
	logsCmd.Flags().BoolVarP(&logsFollow, "follow", "f", false, "follow log output")
	logsCmd.Flags().IntVarP(&logsLines, "lines", "n", 200, "number of lines to show")
	logsCmd.Flags().StringVar(&logsSince, "since", "", "show logs since timestamp")
	logsCmd.Flags().StringVar(&logsUntil, "until", "", "show logs until timestamp")
	logsCmd.Flags().BoolVar(&logsNoColor, "no-color", false, "disable colored output")
}

func runLogs(cmd *cobra.Command, args []string) error {
	job, err := db.FindJobByPartial(args[0])
	if err != nil {
		return fmt.Errorf("failed to find job: %w", err)
	}
	if job == nil {
		return fmt.Errorf("job not found: %s", args[0])
	}

	return systemd.Logs(job.Unit, logsFollow, logsLines, logsSince, logsUntil, logsNoColor)
}
