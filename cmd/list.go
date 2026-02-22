package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"
	"github.com/user/jr/db"
	"github.com/user/jr/systemd"
)

var (
	listLast  int
	listAll   bool
	listState string
	listName  string
	listJSON  bool
)

var listCmd = &cobra.Command{
	Use:     "list [flags]",
	Short:   "List recorded jobs",
	Aliases: []string{"ls", "last"},
	RunE:    runList,
}

func init() {
	listCmd.Flags().IntVar(&listLast, "last", 10, "show last N jobs")
	listCmd.Flags().BoolVar(&listAll, "all", false, "show all jobs")
	listCmd.Flags().StringVar(&listState, "state", "", "filter by state (active, inactive, failed, exited, unknown)")
	listCmd.Flags().StringVar(&listName, "name", "", "filter by name prefix")
	listCmd.Flags().BoolVar(&listJSON, "json", false, "output as JSON")
}

func runList(cmd *cobra.Command, args []string) error {
	var jobs []*db.Job
	var err error

	if listName != "" {
		jobs, err = db.ListJobsByName(listName, listLast)
	} else if listAll {
		jobs, err = db.ListJobs(0, true)
	} else {
		jobs, err = db.ListJobs(listLast, false)
	}

	if err != nil {
		return fmt.Errorf("failed to list jobs: %w", err)
	}

	if len(jobs) == 0 {
		fmt.Println("No jobs found")
		return nil
	}

	units := make([]string, len(jobs))
	for i, job := range jobs {
		units[i] = job.Unit
	}

	unitInfos, err := systemd.ShowUnits(units)
	if err != nil {
		unitInfos = make(map[string]*systemd.UnitInfo)
	}

	if listJSON {
		return outputListJSON(jobs, unitInfos)
	}

	return outputListTable(jobs, unitInfos)
}

func outputListJSON(jobs []*db.Job, unitInfos map[string]*systemd.UnitInfo) error {
	type JobOutput struct {
		ID      int64  `json:"id"`
		Created string `json:"created"`
		Name    string `json:"name"`
		State   string `json:"state"`
		Unit    string `json:"unit"`
		Command string `json:"command"`
	}

	var output []JobOutput
	for _, job := range jobs {
		info := unitInfos[job.Unit]
		state := "unknown"
		if info != nil {
			state = systemd.GetStateString(info)
		}

		var argv []string
		if job.ArgvJSON != "" {
			json.Unmarshal([]byte(job.ArgvJSON), &argv)
		}

		output = append(output, JobOutput{
			ID:      job.ID,
			Created: job.CreatedAtUTC,
			Name:    job.Name,
			State:   state,
			Unit:    job.Unit,
			Command: systemd.ShortenCommand(argv, 40),
		})
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(output)
}

func outputListTable(jobs []*db.Job, unitInfos map[string]*systemd.UnitInfo) error {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tCREATED\tNAME\tSTATE\tUNIT\tCMD")

	for _, job := range jobs {
		info := unitInfos[job.Unit]
		state := "unknown"
		if info != nil {
			state = systemd.GetStateString(info)
		}

		var argv []string
		if job.ArgvJSON != "" {
			json.Unmarshal([]byte(job.ArgvJSON), &argv)
		}

		created, _ := time.Parse(time.RFC3339, job.CreatedAtUTC)
		createdStr := created.Format("Jan 02 15:04")

		stateColored := state
		if isTerminal() {
			switch state {
			case "active":
				stateColored = "\033[32m" + state + "\033[0m"
			case "failed":
				stateColored = "\033[31m" + state + "\033[0m"
			case "exited":
				stateColored = "\033[90m" + state + "\033[0m"
			}
		}

		cmdShort := systemd.ShortenCommand(argv, 30)
		unitShort := job.Unit
		if len(unitShort) > 30 {
			unitShort = unitShort[:27] + "..."
		}

		fmt.Fprintf(w, "%d\t%s\t%s\t%s\t%s\t%s\n",
			job.ID, createdStr, job.Name, stateColored, unitShort, cmdShort)
	}

	return w.Flush()
}

func isTerminal() bool {
	fileInfo, _ := os.Stdout.Stat()
	return (fileInfo.Mode() & os.ModeCharDevice) != 0
}
