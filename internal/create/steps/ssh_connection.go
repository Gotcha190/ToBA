package steps

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/gotcha190/ToBA/internal/create"
)

const (
	remoteWordPressRoot = "www/toba.tamago-dev.pl"
)

var (
	sshTargetPattern = regexp.MustCompile(`^([^\s@]+@[^\s]+)\s+-p\s+([0-9]+)$`)
)

type sshTarget struct {
	UserHost string
	Port     string
}

func parseSSHTarget(raw string) (sshTarget, error) {
	trimmed := strings.TrimSpace(raw)
	match := sshTargetPattern.FindStringSubmatch(trimmed)
	if match == nil {
		return sshTarget{}, fmt.Errorf("invalid TOBA_SSH_TARGET %q; expected format: user@host -p port (example: user@192.168.0.1 -p 22)", raw)
	}

	if _, err := strconv.Atoi(match[2]); err != nil {
		return sshTarget{}, fmt.Errorf("invalid TOBA_SSH_TARGET %q; expected numeric port in format: user@host -p port", raw)
	}

	return sshTarget{
		UserHost: match[1],
		Port:     match[2],
	}, nil
}

func runSSHCommand(ctx *create.Context, target sshTarget, remoteDir string, script string) error {
	return ctx.Runner.Run("", "ssh", "-p", target.Port, target.UserHost, remoteScript(remoteDir, script))
}

func captureSSHCommand(ctx *create.Context, target sshTarget, remoteDir string, script string) (string, error) {
	output, err := ctx.Runner.CaptureOutput("", "ssh", "-p", target.Port, target.UserHost, remoteScript(remoteDir, script))
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(output), nil
}

func copyRemoteFile(ctx *create.Context, target sshTarget, remotePath string, localPath string) error {
	if err := os.MkdirAll(filepath.Dir(localPath), 0755); err != nil {
		return err
	}

	return ctx.Runner.Run("", "scp", "-P", target.Port, target.UserHost+":"+remotePath, localPath)
}

func remoteScript(remoteDir string, script string) string {
	return "cd " + shellQuote(remoteDir) + " && " + script
}

func shellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", `'\''`) + "'"
}
