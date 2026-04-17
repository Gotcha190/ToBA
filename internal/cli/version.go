package cli

import "fmt"

// RunVersion prints the current CLI version string.
//
// Parameters:
// - none
//
// Returns:
// - nothing
//
// Side effects:
// - writes the version string to stdout
//
// Usage:
//
//	toba version
func RunVersion() {
	fmt.Println("toba version: 0.9")
}
