package create

import "sync"

type synchronizedLogger struct {
	next Logger
	mu   sync.Mutex
}

// newSynchronizedLogger wraps next with a mutex-protected logger.
//
// Parameters:
// - next: logger to wrap
//
// Returns:
// - a synchronized logger, nil, or next when it is already synchronized
func newSynchronizedLogger(next Logger) Logger {
	if next == nil {
		return nil
	}

	if _, ok := next.(*synchronizedLogger); ok {
		return next
	}

	return &synchronizedLogger{next: next}
}

// Step writes a synchronized step message.
//
// Parameters:
// - msg: step label to display
//
// Returns:
// - null
//
// Side effects:
// - forwards the message to the wrapped logger
func (l *synchronizedLogger) Step(msg string) {
	l.withLock(func() {
		l.next.Step(msg)
	})
}

// Info writes a synchronized informational message.
//
// Parameters:
// - msg: informational message to display
//
// Returns:
// - null
//
// Side effects:
// - forwards the message to the wrapped logger
func (l *synchronizedLogger) Info(msg string) {
	l.withLock(func() {
		l.next.Info(msg)
	})
}

// Prompt writes a synchronized prompt message.
//
// Parameters:
// - msg: prompt text to display
//
// Returns:
// - null
//
// Side effects:
// - forwards the message to the wrapped logger
func (l *synchronizedLogger) Prompt(msg string) {
	l.withLock(func() {
		l.next.Prompt(msg)
	})
}

// Warning writes a synchronized warning message.
//
// Parameters:
// - msg: warning message to display
//
// Returns:
// - null
//
// Side effects:
// - forwards the message to the wrapped logger
func (l *synchronizedLogger) Warning(msg string) {
	l.withLock(func() {
		l.next.Warning(msg)
	})
}

// Success writes a synchronized success message.
//
// Parameters:
// - msg: success message to display
//
// Returns:
// - null
//
// Side effects:
// - forwards the message to the wrapped logger
func (l *synchronizedLogger) Success(msg string) {
	l.withLock(func() {
		l.next.Success(msg)
	})
}

// Error writes a synchronized error message.
//
// Parameters:
// - msg: error message to display
//
// Returns:
// - null
//
// Side effects:
// - forwards the message to the wrapped logger
func (l *synchronizedLogger) Error(msg string) {
	l.withLock(func() {
		l.next.Error(msg)
	})
}

// ErrorCode writes a synchronized coded error message.
//
// Parameters:
// - code: symbolic error code
// - msg: human-readable error message
//
// Returns:
// - null
//
// Side effects:
// - forwards the message to the wrapped logger
func (l *synchronizedLogger) ErrorCode(code string, msg string) {
	l.withLock(func() {
		l.next.ErrorCode(code, msg)
	})
}

// withLock runs fn while holding the logger mutex.
//
// Parameters:
// - fn: callback to execute while locked
//
// Returns:
// - null
//
// Side effects:
// - serializes access to the wrapped logger
func (l *synchronizedLogger) withLock(fn func()) {
	l.mu.Lock()
	defer l.mu.Unlock()

	fn()
}
