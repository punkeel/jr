package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/user/jr/systemd"
)

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Check environment and prerequisites",
	RunE:  runDoctor,
}

func runDoctor(cmd *cobra.Command, args []string) error {
	fmt.Println("Checking environment...")
	fmt.Println()

	allOK := true

	fmt.Print("systemd user instance: ")
	if err := systemd.CheckUserSystemd(); err != nil {
		fmt.Println("FAIL")
		fmt.Printf("  Error: %v\n", err)
		allOK = false
	} else {
		fmt.Println("OK")
	}

	fmt.Print("systemd-run: ")
	if err := systemd.CheckSystemdRun(); err != nil {
		fmt.Println("FAIL")
		fmt.Println("  systemd-run not found in PATH")
		fmt.Println("  Install systemd package")
		allOK = false
	} else {
		fmt.Println("OK")
	}

	fmt.Print("journalctl: ")
	if err := systemd.CheckJournalctl(); err != nil {
		fmt.Println("FAIL")
		fmt.Println("  journalctl not found in PATH")
		allOK = false
	} else {
		fmt.Println("OK")
	}

	fmt.Print("lingering: ")
	linger, err := systemd.CheckLingering()
	if err != nil {
		fmt.Println("UNKNOWN")
		fmt.Printf("  Error checking: %v\n", err)
	} else if linger {
		fmt.Println("OK (enabled)")
	} else {
		fmt.Println("WARNING (not enabled)")
		fmt.Printf("  Jobs may stop when you log out.\n")
		fmt.Printf("  To enable: sudo loginctl enable-linger %s\n", os.Getenv("USER"))
		allOK = false
	}

	fmt.Println()
	if allOK {
		fmt.Println("All checks passed!")
	} else {
		fmt.Println("Some checks failed. See above for details.")
	}

	return nil
}
