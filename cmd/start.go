package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/user/jr/db"
	"github.com/user/jr/systemd"
)

var (
	startName          string
	startCwd           string
	startEnv           []string
	startDesc          string
	startGPU           string
	startNoLingerCheck bool
	startProperties    []string
)

var startCmd = &cobra.Command{
	Use:   "start [flags] -- <command> [args...]",
	Short: "Start a new job",
	Long:  `Start a new job via systemd-run. The job will continue running after disconnect.`,
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) < 1 {
			return fmt.Errorf("requires a command to run")
		}
		return nil
	},
	RunE: runStart,
}

func init() {
	startCmd.Flags().StringVarP(&startName, "name", "n", "", "logical name (default: derived from command)")
	startCmd.Flags().StringVar(&startCwd, "cwd", "", "working directory (default: current)")
	startCmd.Flags().StringArrayVarP(&startEnv, "env", "e", nil, "environment variables (repeatable, format: K=V)")
	startCmd.Flags().StringVar(&startDesc, "desc", "", "override unit description")
	startCmd.Flags().StringVar(&startGPU, "gpu", "", "convenience: sets CUDA_VISIBLE_DEVICES=<idx>")
	startCmd.Flags().BoolVar(&startNoLingerCheck, "no-linger-check", false, "skip linger hint if not enabled")
	startCmd.Flags().StringArrayVar(&startProperties, "property", nil, "pass -p k=v to systemd-run (repeatable)")
}

func runStart(cmd *cobra.Command, args []string) error {
	command := args[0]
	argv := args

	if !systemd.CommandExists(command) {
		return fmt.Errorf("command not found: %s", command)
	}

	name := startName
	if name == "" {
		name = filepath.Base(command)
	}

	cwd := startCwd
	if cwd == "" {
		var err error
		cwd, err = os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get current directory: %w", err)
		}
	}

	env := make(map[string]string)
	for _, e := range startEnv {
		parts := strings.SplitN(e, "=", 2)
		if len(parts) != 2 {
			return fmt.Errorf("invalid env format: %s (expected K=V)", e)
		}
		env[parts[0]] = parts[1]
	}

	if startGPU != "" {
		env["CUDA_VISIBLE_DEVICES"] = startGPU
	}

	props := make(map[string]string)
	for _, p := range startProperties {
		parts := strings.SplitN(p, "=", 2)
		if len(parts) != 2 {
			return fmt.Errorf("invalid property format: %s (expected k=v)", p)
		}
		props[parts[0]] = parts[1]
	}

	unit := systemd.GenerateUnitName(name)
	desc := startDesc
	if desc == "" {
		desc = fmt.Sprintf("jr job: %s", name)
	}

	if !startNoLingerCheck {
		linger, err := systemd.CheckLingering()
		if err == nil && !linger {
			fmt.Fprintf(os.Stderr, "Warning: lingering not enabled. Jobs may stop on logout.\n")
			fmt.Fprintf(os.Stderr, "Enable with: sudo loginctl enable-linger $USER\n\n")
		}
	}

	if err := systemd.StartUnit(unit, cwd, argv, env, props, desc); err != nil {
		return fmt.Errorf("failed to start unit: %w", err)
	}

	host, _ := os.Hostname()
	user := os.Getenv("USER")

	id, err := db.CreateJob(name, unit, cwd, argv, env, props, host, user)
	if err != nil {
		return fmt.Errorf("job started but failed to record: %w", err)
	}

	fmt.Printf("Started %d %s\n", id, unit)
	return nil
}
