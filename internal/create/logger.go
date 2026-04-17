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

func (l ConsoleLogger) Step(msg string) {
	l.println("[STEP]", msg)
}

func (l ConsoleLogger) Info(msg string) {
	l.println("[INFO]", msg)
}

func (l ConsoleLogger) Prompt(msg string) {
	fmt.Fprint(l.writer(), "[PROMPT] ", msg)
}

func (l ConsoleLogger) Success(msg string) {
	l.println("[OK]", msg)
}

func (l ConsoleLogger) Warning(msg string) {
	l.println("[WARNING]", msg)
}

func (l ConsoleLogger) Error(msg string) {
	l.println("[ERROR]", msg)
}

func (l ConsoleLogger) ErrorCode(code string, msg string) {
	l.println(fmt.Sprintf("[ERROR][%s]", code), msg)
}

func NewConsoleLogger(out io.Writer) Logger {
	return ConsoleLogger{Out: out}
}

func (l ConsoleLogger) println(prefix string, msg string) {
	fmt.Fprintln(l.writer(), prefix, msg)
}

func (l ConsoleLogger) writer() io.Writer {
	if l.Out != nil {
		return l.Out
	}
	return os.Stdout
}
