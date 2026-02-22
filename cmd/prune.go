package cmd

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"
	"github.com/user/jr/db"
)

var (
	pruneKeep       int
	pruneOlderThan  string
	pruneFailedOnly bool
)

var pruneCmd = &cobra.Command{
	Use:   "prune [flags]",
	Short: "Remove old jobs from the registry",
	RunE:  runPrune,
}

func init() {
	pruneCmd.Flags().IntVar(&pruneKeep, "keep", 100, "keep last N jobs")
	pruneCmd.Flags().StringVar(&pruneOlderThan, "older-than", "", "remove jobs older than duration (e.g., 7d, 24h)")
	pruneCmd.Flags().BoolVar(&pruneFailedOnly, "failed-only", false, "only remove failed jobs")
}

func runPrune(cmd *cobra.Command, args []string) error {
	var duration time.Duration
	if pruneOlderThan != "" {
		var err error
		duration, err = parseDuration(pruneOlderThan)
		if err != nil {
			return fmt.Errorf("invalid duration: %w", err)
		}
	}

	if err := db.PruneJobs(pruneKeep, duration, pruneFailedOnly); err != nil {
		return fmt.Errorf("failed to prune jobs: %w", err)
	}

	fmt.Printf("Pruned old jobs (keeping last %d)\n", pruneKeep)
	return nil
}

func parseDuration(s string) (time.Duration, error) {
	// Simple duration parsing supporting d, h, m suffixes
	var n int
	var unit string
	_, err := fmt.Sscanf(s, "%d%s", &n, &unit)
	if err != nil {
		return time.ParseDuration(s)
	}

	switch unit {
	case "d":
		return time.Duration(n) * 24 * time.Hour, nil
	case "h":
		return time.Duration(n) * time.Hour, nil
	case "m":
		return time.Duration(n) * time.Minute, nil
	default:
		return time.ParseDuration(s)
	}
}
