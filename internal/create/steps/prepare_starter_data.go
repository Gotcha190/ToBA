package steps

import (
	"fmt"
	"os"
	"strings"

	"github.com/gotcha190/toba/internal/create"
)

const (
	starterDataModeLocal  = "local"
	starterDataModeRemote = "remote"
)

type PrepareStarterDataStep struct{}

func NewPrepareStarterDataStep() *PrepareStarterDataStep {
	return &PrepareStarterDataStep{}
}

func (s *PrepareStarterDataStep) Name() string {
	return "Prepare starter data"
}

func (s *PrepareStarterDataStep) Run(ctx *create.Context) error {
	rootInfo, err := os.Stat(ctx.Paths.Root)
	switch {
	case err == nil && rootInfo.IsDir():
		return prepareLocalStarterData(ctx)
	case err == nil:
		return fmt.Errorf("project path exists and is not a directory: %s", ctx.Paths.Root)
	case !os.IsNotExist(err):
		return err
	default:
		if strings.TrimSpace(ctx.Config.SSHTarget) == "" {
			globalEnvPath, pathErr := create.GlobalEnvPath()
			if pathErr != nil {
				return fmt.Errorf("SSH starter source is not configured; set TOBA_SSH_TARGET in the global config or pass --ssh-target")
			}
			return fmt.Errorf("SSH starter source is not configured; fill in TOBA_SSH_TARGET in %s or pass --ssh-target", globalEnvPath)
		}
		return prepareRemoteStarterData(ctx)
	}
}
