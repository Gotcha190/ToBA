package cmd

import (
	"flag"
	"fmt"
	"os"

	"github.com/gotcha190/toba/internal/cli"
)

const usageBanner = `
░▒▓████████▓▒░▒▓██████▓▒░░▒▓███████▓▒░ ░▒▓██████▓▒░
   ░▒▓█▓▒░  ░▒▓█▓▒░░▒▓█▓▒░▒▓█▓▒░░▒▓█▓▒░▒▓█▓▒░░▒▓█▓▒░
   ░▒▓█▓▒░  ░▒▓█▓▒░░▒▓█▓▒░▒▓█▓▒░░▒▓█▓▒░▒▓█▓▒░░▒▓█▓▒░
   ░▒▓█▓▒░  ░▒▓█▓▒░░▒▓█▓▒░▒▓███████▓▒░░▒▓████████▓▒░
   ░▒▓█▓▒░  ░▒▓█▓▒░░▒▓█▓▒░▒▓█▓▒░░▒▓█▓▒░▒▓█▓▒░░▒▓█▓▒░
   ░▒▓█▓▒░  ░▒▓█▓▒░░▒▓█▓▒░▒▓█▓▒░░▒▓█▓▒░▒▓█▓▒░░▒▓█▓▒░
   ░▒▓█▓▒░   ░▒▓██████▓▒░░▒▓███████▓▒░░▒▓█▓▒░░▒▓█▓▒░
`

// Execute parses the root command name from os.Args and dispatches execution
// to the matching CLI handler.
//
// Returns:
// - null
//
// Side effects:
//   - writes usage text to stdout when no command is provided
//   - writes command errors to stderr
//   - may terminate the process with a non-zero exit code for invalid commands
//     or command failures
//
// Usage:
//
//	toba <command>
func Execute() {
	if len(os.Args) < 2 {
		printUsage()
		return
	}

	switch os.Args[1] {
	case "create":
		if err := runCreate(os.Args[2:]); err != nil {
			fmt.Fprintln(os.Stderr, "Error:", err)
			os.Exit(1)
		}
	case "config":
		if err := runConfig(os.Args[2:]); err != nil {
			fmt.Fprintln(os.Stderr, "Error:", err)
			os.Exit(1)
		}
	case "doctor":
		if err := cli.RunDoctor(); err != nil {
			fmt.Fprintln(os.Stderr, "Error:", err)
			os.Exit(1)
		}
	case "version":
		cli.RunVersion()
	default:
		printUsage()
		os.Exit(1)
	}
}

// runCreate parses arguments for the `toba create` command and forwards the
// normalized options to the create CLI entrypoint.
//
// Parameters:
// - args: raw command-line arguments after the `create` subcommand
//
// Returns:
//   - an error when flag parsing fails, required arguments are missing, or
//     unexpected positional arguments remain after parsing
//
// Side effects:
// - configures a dedicated flag set that writes parsing errors to stderr
//
// Usage:
//
//	toba create demo --php=8.4 --starter-repo=git@github.com:org/repo.git --ssh-target='user@host -p 22' --remote-wordpress-root='www/example.com' --dry-run
func runCreate(args []string) error {
	fs := flag.NewFlagSet("create", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)

	var opts cli.CreateOptions
	fs.StringVar(&opts.PHPVersion, "php", "", "PHP version for the Lando appserver")
	fs.StringVar(&opts.StarterRepo, "starter-repo", "", "Git repository for the starter theme")
	fs.StringVar(&opts.SSHTarget, "ssh-target", "", "SSH target in format 'user@host -p port'")
	fs.StringVar(&opts.RemoteWordPressRoot, "remote-wordpress-root", "", "Remote WordPress root used by SSH starter data")
	fs.BoolVar(&opts.DryRun, "dry-run", false, "Print planned actions without writing files")
	fs.Usage = func() {
		fmt.Fprintln(os.Stderr, "Usage: toba create [project-name] [--php=8.4] [--starter-repo=git@github.com:org/repo.git] [--ssh-target='user@host -p port'] [--remote-wordpress-root='www/example.com'] [--dry-run]")
	}

	projectName := ""
	if len(args) > 0 && args[0] != "" && args[0][0] != '-' {
		projectName = args[0]
		args = args[1:]
	}

	if err := fs.Parse(args); err != nil {
		return err
	}

	remaining := fs.Args()
	if projectName == "" && len(remaining) > 0 {
		projectName = remaining[0]
		remaining = remaining[1:]
	}

	if len(remaining) > 0 {
		return fmt.Errorf("unexpected arguments: %v", remaining)
	}

	opts.Name = projectName
	return cli.RunCreate(opts)
}

// runConfig validates arguments for the `toba config` command and runs the
// config bootstrap flow without any nested subcommands.
//
// Parameters:
// - args: raw command-line arguments after the `config` subcommand
//
// Returns:
// - an error when unexpected arguments are provided
//
// Side effects:
// - triggers global configuration initialization through the CLI layer
//
// Usage:
//
//	toba config
func runConfig(args []string) error {
	if len(args) > 0 {
		return fmt.Errorf("usage: toba config")
	}

	return cli.RunConfig()
}

// printUsage prints the ASCII banner, the root usage line, and the list of
// available CLI commands.
//
// Returns:
// - null
//
// Side effects:
// - writes formatted help text to stdout
func printUsage() {
	fmt.Print(usageBanner)
	fmt.Println()
	fmt.Println("Usage: toba <command>")
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println("  config   Initialize global configuration")
	fmt.Println("  create   Create a new project skeleton")
	fmt.Println("  doctor   Check system dependencies")
	fmt.Println("  version  Print the current version")
}
