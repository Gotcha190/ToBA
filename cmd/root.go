package cmd

import (
	"flag"
	"fmt"
	"os"

	"github.com/gotcha190/ToBA/internal/cli"
)

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

func runCreate(args []string) error {
	fs := flag.NewFlagSet("create", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)

	var opts cli.CreateOptions
	fs.StringVar(&opts.PHPVersion, "php", "", "PHP version for the Lando appserver")
	fs.StringVar(&opts.Domain, "domain", "", "Local domain for the project")
	fs.StringVar(&opts.StarterRepo, "starter-repo", "", "Git repository for the starter theme")
	fs.StringVar(&opts.SSHTarget, "ssh-target", "", "SSH target in format 'user@host -p port'")
	fs.BoolVar(&opts.DryRun, "dry-run", false, "Print planned actions without writing files")
	fs.Usage = func() {
		fmt.Fprintln(os.Stderr, "Usage: toba create [project-name] [--php=8.4] [--domain=project.lndo.site] [--starter-repo=git@github.com:org/repo.git] [--ssh-target='user@host -p port'] [--dry-run]")
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

func runConfig(args []string) error {
	if len(args) != 1 || args[0] != "init" {
		return fmt.Errorf("usage: toba config init")
	}

	return cli.RunConfigInit()
}

func printUsage() {
	fmt.Println("Usage: toba <command>")
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println("  config   Initialize global ToBA configuration")
	fmt.Println("  create   Create a new project skeleton")
	fmt.Println("  doctor   Check system dependencies")
	fmt.Println("  version  Print the current version")
}
