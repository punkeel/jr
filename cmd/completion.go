package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

var completionCmd = &cobra.Command{
	Use:   "completion [bash|zsh|fish|powershell]",
	Short: "Generate completion script",
	Long: `To load completions:

Bash:

  $ source <(jr completion bash)

  # To load completions for each session, execute once:
  # Linux:
  $ jr completion bash > /etc/bash_completion.d/jr
  # macOS:
  $ jr completion bash > /usr/local/etc/bash_completion.d/jr

Zsh:

  # If shell completion is not already enabled in your environment,
  # you will need to enable it.  You can execute the following once:

  $ echo "autoload -U compinit; compinit" >> ~/.zshrc

  # To load completions for each session, execute once:
  $ jr completion zsh > "${fpath[1]}/_jr"

  # You will need to start a new shell for this setup to take effect.

Fish:

  $ jr completion fish | source

  # To load completions for each session, execute once:
  $ jr completion fish > ~/.config/fish/completions/jr.fish

PowerShell:

  PS> jr completion powershell | Out-String | Invoke-Expression

  # To load completions for every new session, run:
  PS> jr completion powershell > jr.ps1
  # and source this file from your PowerShell profile.
`,
	DisableFlagsInUseLine: true,
	ValidArgs:             []string{"bash", "zsh", "fish", "powershell"},
	Args:                  cobra.ExactValidArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		switch args[0] {
		case "bash":
			rootCmd.GenBashCompletion(os.Stdout)
		case "zsh":
			rootCmd.GenZshCompletion(os.Stdout)
		case "fish":
			rootCmd.GenFishCompletion(os.Stdout, true)
		case "powershell":
			rootCmd.GenPowerShellCompletionWithDesc(os.Stdout)
		}
	},
}
