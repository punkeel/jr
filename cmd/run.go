package cmd

import (
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/spf13/cobra"
	"github.com/user/jr/db"
	"github.com/user/jr/systemd"
)

// completeCommands returns executable names from PATH matching the prefix
func completeCommands(prefix string) []string {
	var matches []string
	seen := make(map[string]bool)

	pathEnv := os.Getenv("PATH")
	for _, dir := range filepath.SplitList(pathEnv) {
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			name := entry.Name()
			if strings.HasPrefix(name, prefix) && !seen[name] {
				// Check if executable
				info, err := entry.Info()
				if err != nil {
					continue
				}
				if info.Mode()&0111 != 0 {
					matches = append(matches, name)
					seen[name] = true
				}
			}
		}
	}
	return matches
}

var (
	runName          string
	runCwd           string
	runEnv           []string
	runDesc          string
	runGPU           string
	runNoLingerCheck bool
	runProperties    []string
	runAttach        bool
)

var runCmd = &cobra.Command{
	Use:     "run [flags] -- <command> [args...]",
	Aliases: []string{"start"},
	Short:   "Run a new job",
	Long:    `Run a new job via systemd-run. The job will continue running after disconnect.`,
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) < 1 {
			return fmt.Errorf("requires a command to run")
		}
		return nil
	},
	ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		// For the first argument, complete commands from PATH
		// For subsequent arguments, let shell provide default (file) completion
		if len(args) == 0 {
			return completeCommands(toComplete), cobra.ShellCompDirectiveNoFileComp
		}
		return nil, cobra.ShellCompDirectiveDefault
	},
	RunE: runRun,
}

func init() {
	runCmd.Flags().StringVarP(&runName, "name", "n", "", "logical name (default: derived from command)")
	runCmd.Flags().StringVar(&runCwd, "cwd", "", "working directory (default: current)")
	runCmd.Flags().StringArrayVarP(&runEnv, "env", "e", nil, "environment variables (repeatable, format: K=V)")
	runCmd.Flags().StringVar(&runDesc, "desc", "", "override unit description")
	runCmd.Flags().StringVar(&runGPU, "gpu", "", "convenience: sets CUDA_VISIBLE_DEVICES=<idx>")
	runCmd.Flags().BoolVar(&runNoLingerCheck, "no-linger-check", false, "skip linger hint if not enabled")
	runCmd.Flags().StringArrayVar(&runProperties, "property", nil, "pass -p k=v to systemd-run (repeatable)")
	runCmd.Flags().BoolVarP(&runAttach, "attach", "a", false, "attach to job output (ctrl+c detaches, job keeps running)")
}

func runRun(cmd *cobra.Command, args []string) error {
	command := args[0]
	argv := args

	if !systemd.CommandExists(command) {
		return fmt.Errorf("command not found: %s", command)
	}

	name := runName
	if name == "" {
		name = filepath.Base(command)
	}

	cwd := runCwd
	if cwd == "" {
		var err error
		cwd, err = os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get current directory: %w", err)
		}
	}

	env := make(map[string]string)
	for _, e := range os.Environ() {
		parts := strings.SplitN(e, "=", 2)
		if len(parts) != 2 {
			continue
		}
		env[parts[0]] = parts[1]
	}
	for _, e := range runEnv {
		parts := strings.SplitN(e, "=", 2)
		if len(parts) != 2 {
			return fmt.Errorf("invalid env format: %s (expected K=V)", e)
		}
		env[parts[0]] = parts[1]
	}

	if runGPU != "" {
		env["CUDA_VISIBLE_DEVICES"] = runGPU
	}

	props := make(map[string]string)
	for _, p := range runProperties {
		parts := strings.SplitN(p, "=", 2)
		if len(parts) != 2 {
			return fmt.Errorf("invalid property format: %s (expected k=v)", p)
		}
		props[parts[0]] = parts[1]
	}

	// Set up colored output if in attach mode
	if runAttach {
		// Enable color output in systemd journal
		props["StandardOutput"] = "journal+console"
		props["StandardError"] = "journal+console"
		// Preserve TERM and COLORTERM for color support
		if term := os.Getenv("TERM"); term != "" {
			env["TERM"] = term
		}
		if colorterm := os.Getenv("COLORTERM"); colorterm != "" {
			env["COLORTERM"] = colorterm
		}
		// Force color for common tools
		env["FORCE_COLOR"] = "1"
		env["CLICOLOR_FORCE"] = "1"
	}

	unit := systemd.GenerateUnitName(name)
	desc := runDesc
	if desc == "" {
		desc = fmt.Sprintf("jr job: %s", name)
	}

	if !runNoLingerCheck {
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

	// If attach mode, stream logs until interrupted
	if runAttach {
		fmt.Println()
		fmt.Println("=== Attached to job output (press Ctrl+C to detach, job continues running) ===")
		fmt.Println()

		// Set up signal handler for graceful detach
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

		// Start log streaming in background
		logDone := make(chan error, 1)
		go func() {
			logDone <- systemd.Logs(unit, true, 0, "", "", false, false)
		}()

		// Wait for either signal or log completion
		select {
		case <-sigChan:
			fmt.Println()
			fmt.Println("=== Detached from job (job is still running) ===")
			fmt.Printf("View logs: jr logs %d\n", id)
			fmt.Printf("Stop job:  jr stop %d\n", id)
			return nil
		case err := <-logDone:
			if err != nil {
				return fmt.Errorf("log stream ended: %w", err)
			}
			return nil
		}
	}

	return nil
}
