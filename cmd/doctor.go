package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/user/jr/systemd"
)

const (
	colorReset  = "\033[0m"
	colorRed    = "\033[31m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorCyan   = "\033[36m"
	colorBold   = "\033[1m"
)

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Check environment and prerequisites",
	RunE:  runDoctor,
}

func runDoctor(cmd *cobra.Command, args []string) error {
	useColor := isTerminal()

	fmt.Println("Checking environment...")
	fmt.Println()

	allOK := true

	fmt.Print("systemd user instance: ")
	if err := systemd.CheckUserSystemd(); err != nil {
		if useColor {
			fmt.Printf("%sFAIL%s\n", colorRed, colorReset)
		} else {
			fmt.Println("FAIL")
		}
		fmt.Printf("  Error: %v\n", err)
		allOK = false
	} else {
		if useColor {
			fmt.Printf("%sOK%s\n", colorGreen, colorReset)
		} else {
			fmt.Println("OK")
		}
	}

	fmt.Print("systemd-run: ")
	if err := systemd.CheckSystemdRun(); err != nil {
		if useColor {
			fmt.Printf("%sFAIL%s\n", colorRed, colorReset)
		} else {
			fmt.Println("FAIL")
		}
		fmt.Println("  systemd-run not found in PATH")
		fmt.Println("  Install systemd package")
		allOK = false
	} else {
		if useColor {
			fmt.Printf("%sOK%s\n", colorGreen, colorReset)
		} else {
			fmt.Println("OK")
		}
	}

	fmt.Print("journalctl: ")
	if err := systemd.CheckJournalctl(); err != nil {
		if useColor {
			fmt.Printf("%sFAIL%s\n", colorRed, colorReset)
		} else {
			fmt.Println("FAIL")
		}
		fmt.Println("  journalctl not found in PATH")
		allOK = false
	} else {
		if useColor {
			fmt.Printf("%sOK%s\n", colorGreen, colorReset)
		} else {
			fmt.Println("OK")
		}
	}

	fmt.Print("lingering: ")
	linger, err := systemd.CheckLingering()
	if err != nil {
		if useColor {
			fmt.Printf("%sUNKNOWN%s\n", colorYellow, colorReset)
		} else {
			fmt.Println("UNKNOWN")
		}
		fmt.Printf("  Error checking: %v\n", err)
	} else if linger {
		if useColor {
			fmt.Printf("%sOK (enabled)%s\n", colorGreen, colorReset)
		} else {
			fmt.Println("OK (enabled)")
		}
	} else {
		if useColor {
			fmt.Printf("%sWARNING (not enabled)%s\n", colorYellow, colorReset)
			fmt.Printf("  %sJobs may stop when you log out.%s\n", colorYellow, colorReset)
			fmt.Printf("  To enable: %ssudo loginctl enable-linger %s%s\n", colorCyan, os.Getenv("USER"), colorReset)
		} else {
			fmt.Println("WARNING (not enabled)")
			fmt.Printf("  Jobs may stop when you log out.\n")
			fmt.Printf("  To enable: sudo loginctl enable-linger %s\n", os.Getenv("USER"))
		}
		allOK = false
	}

	fmt.Println()
	if allOK {
		if useColor {
			fmt.Printf("%s%sAll checks passed!%s\n", colorBold, colorGreen, colorReset)
		} else {
			fmt.Println("All checks passed!")
		}
	} else {
		if useColor {
			fmt.Printf("%s%sSome checks failed. See above for details.%s\n", colorBold, colorRed, colorReset)
		} else {
			fmt.Println("Some checks failed. See above for details.")
		}
	}

	return nil
}
