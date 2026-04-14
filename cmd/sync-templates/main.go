package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/gotcha190/ToBA/internal/templatesync"
)

func main() {
	repoRoot, err := os.Getwd()
	if err != nil {
		fail(err)
	}

	repoRoot, err = filepath.Abs(repoRoot)
	if err != nil {
		fail(err)
	}

	if err := templatesync.SyncRepo(repoRoot); err != nil {
		fail(err)
	}

	fmt.Println("Synced templates -> internal/templates/files")
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
