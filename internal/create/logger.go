package create

import (
	"fmt"
	"io"
	"os"
)

type Logger interface {
	Step(msg string)
	Info(msg string)
	Prompt(msg string)
	Warning(msg string)
	Success(msg string)
	Error(msg string)
	ErrorCode(code string, msg string)
}

type ConsoleLogger struct {
	Out io.Writer
}

// Step writes a formatted step marker to the configured output.
//
// Parameters:
// - msg: step label to display
//
// Returns:
// - null
//
// Side effects:
// - writes a single formatted line to the logger output
func (l ConsoleLogger) Step(msg string) {
	l.println("[STEP]", msg)
}

// Info writes an informational log message to the configured output.
//
// Parameters:
// - msg: informational message to display
//
// Returns:
// - null
func (l ConsoleLogger) Info(msg string) {
	l.println("[INFO]", msg)
}

// Prompt writes a prompt without appending a trailing newline.
//
// Parameters:
// - msg: prompt text to display
//
// Returns:
// - null
func (l ConsoleLogger) Prompt(msg string) {
	_, _ = fmt.Fprint(l.writer(), "[PROMPT] ", msg)
}

// Success writes a success message to the configured output.
//
// Parameters:
// - msg: success message to display
//
// Returns:
// - null
func (l ConsoleLogger) Success(msg string) {
	l.println("[OK]", msg)
}

// Warning writes a warning message to the configured output.
//
// Parameters:
// - msg: warning message to display
//
// Returns:
// - null
func (l ConsoleLogger) Warning(msg string) {
	l.println("[WARNING]", msg)
}

// Error writes an error message to the configured output.
//
// Parameters:
// - msg: error message to display
//
// Returns:
// - null
func (l ConsoleLogger) Error(msg string) {
	l.println("[ERROR]", msg)
}

// ErrorCode writes an error message that includes a machine-readable code.
//
// Parameters:
// - code: symbolic error code
// - msg: human-readable error message
//
// Returns:
// - null
func (l ConsoleLogger) ErrorCode(code string, msg string) {
	l.println(fmt.Sprintf("[ERROR][%s]", code), msg)
}

// NewConsoleLogger creates a Logger implementation that writes to out.
//
// Parameters:
// - out: optional writer used for all log output
//
// Returns:
// - a Logger backed by ConsoleLogger
func NewConsoleLogger(out io.Writer) Logger {
	return ConsoleLogger{Out: out}
}

// println writes a prefixed line using the logger's resolved writer.
//
// Parameters:
// - prefix: log prefix such as [INFO] or [ERROR]
// - msg: message body to print
//
// Returns:
// - null
func (l ConsoleLogger) println(prefix string, msg string) {
	_, _ = fmt.Fprintln(l.writer(), prefix, msg)
}

// writer returns the configured output writer or stdout when none is set.
//
// Returns:
// - the configured writer, or os.Stdout as a fallback
func (l ConsoleLogger) writer() io.Writer {
	if l.Out != nil {
		return l.Out
	}
	return os.Stdout
}
