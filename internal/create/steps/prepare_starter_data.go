package steps

import (
	"fmt"
	"os"

	"github.com/gotcha190/ToBA/internal/create"
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
		return prepareRemoteStarterData(ctx)
	}
}
