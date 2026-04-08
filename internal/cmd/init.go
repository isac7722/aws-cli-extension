package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:       "init [bash|zsh]",
	Short:     "Output shell wrapper script for AWS_PROFILE switching",
	Long:      `Outputs a shell wrapper script that enables AWS_PROFILE switching via awse. Add 'eval "$(awse init bash)"' or 'eval "$(awse init zsh)"' to your shell profile.`,
	Args:      cobra.ExactArgs(1),
	ValidArgs: []string{"bash", "zsh"},
	RunE:      runInit,
}

func init() {
	rootCmd.AddCommand(initCmd)
}

func runInit(cmd *cobra.Command, args []string) error {
	shell := args[0]
	switch shell {
	case "bash", "zsh":
		fmt.Fprint(cmd.OutOrStdout(), shellWrapper(shell))
		return nil
	default:
		return fmt.Errorf("unsupported shell: %s (supported: bash, zsh)", shell)
	}
}

func shellWrapper(shell string) string {
	// The wrapper function intercepts 'awse user switch' to capture the exported
	// AWS_PROFILE from the underlying binary's output. All other subcommands
	// pass through directly.
	//
	// Protocol: when 'awse user switch' succeeds, the binary prints a line
	// "AWSE_EXPORT:AWS_PROFILE=<profile>" to stdout. The wrapper captures that
	// line, exports the variable, and strips it from visible output.
	return `# awse shell wrapper — add 'eval "$(awse init ` + shell + `)"' to your shell profile
awse() {
  local awse_bin
  awse_bin="$(command -v awse)"
  if [ -z "$awse_bin" ]; then
    echo "awse: binary not found in PATH" >&2
    return 1
  fi

  case "$1" in
    user)
      case "$2" in
        switch)
          local output
          output="$("$awse_bin" "$@" 2>&1)"
          local exit_code=$?
          if [ $exit_code -ne 0 ]; then
            echo "$output" >&2
            return $exit_code
          fi
          # Parse AWSE_EXPORT lines and export variables
          local line
          while IFS= read -r line; do
            case "$line" in
              AWSE_EXPORT:*)
                local assignment="${line#AWSE_EXPORT:}"
                local key="${assignment%%=*}"
                local value="${assignment#*=}"
                if [ -n "$key" ]; then
                  export "$key"="$value"
                fi
                ;;
              *)
                [ -n "$line" ] && echo "$line"
                ;;
            esac
          done <<< "$output"
          return 0
          ;;
        *)
          command "$awse_bin" "$@"
          return $?
          ;;
      esac
      ;;
    *)
      command "$awse_bin" "$@"
      return $?
      ;;
  esac
}
`
}
