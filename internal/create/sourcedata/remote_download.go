package sourcedata

import (
	"fmt"
	"sync"

	"github.com/gotcha190/toba/internal/create"
)

type remoteDownload struct {
	name       string
	remotePath string
	localPath  string
}

// downloadRemoteFiles copies prepared starter files from the SSH host in parallel.
//
// Parameters:
// - ctx: shared create context containing runner and logger state
// - target: parsed SSH destination
// - downloads: remote-to-local artifact copy definitions
//
// Returns:
// - an error when any required download fails
func downloadRemoteFiles(ctx *create.Context, target sshTarget, downloads []remoteDownload) error {
	var once sync.Once
	var runErr error
	var wg sync.WaitGroup

	for _, download := range downloads {
		download := download
		wg.Add(1)
		go func() {
			defer wg.Done()

			if err := copyRemoteFile(ctx, target, download.remotePath, download.localPath); err != nil {
				once.Do(func() {
					runErr = fmt.Errorf("failed to download starter %s over SSH: %w", download.name, err)
				})
			}
		}()
	}

	wg.Wait()
	return runErr
}
