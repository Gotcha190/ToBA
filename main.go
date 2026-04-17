package main

import "github.com/gotcha190/toba/cmd"

// main is the process entrypoint for the ToBA binary.
//
// It delegates all CLI parsing and command dispatch to the cmd package.
//
// Parameters:
// - none
//
// Returns:
// - nothing
//
// Side effects:
//   - executes the selected CLI command and may terminate the process through
//     downstream command handling
func main() {
	cmd.Execute()
}
