package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/user/jr/db"
)

var rootCmd = &cobra.Command{
	Use:   "jr",
	Short: "jr - Job Runner: manage long-running jobs via systemd",
	Long: `jr (job run) is a CLI tool for starting, monitoring, and managing
long-running jobs via systemd user units. Jobs survive SSH disconnects
and can be monitored from any session.`,
	SilenceErrors: false,
	SilenceUsage:  true,
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) > 0 {
			return fmt.Errorf("unknown command: %s\nSee 'jr --help' for available commands", args[0])
		}
		return cmd.Help()
	},
}

func Execute() error {
	defer db.Close()
	return rootCmd.Execute()
}

func init() {
	rootCmd.AddCommand(runCmd)
	rootCmd.AddCommand(listCmd)
	rootCmd.AddCommand(statusCmd)
	rootCmd.AddCommand(logsCmd)
	rootCmd.AddCommand(stopCmd)
	rootCmd.AddCommand(rmCmd)
	rootCmd.AddCommand(pruneCmd)
	rootCmd.AddCommand(doctorCmd)
	rootCmd.AddCommand(completionCmd)

	cobra.OnInitialize(initDB)
}

func initDB() {
	if err := db.InitDB(); err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing database: %v\n", err)
		os.Exit(1)
	}
}
