package create

import (
	"fmt"
	"time"
)

// FormatDurationCompact formats a duration for concise CLI output.
//
// Parameters:
// - d: duration to format
//
// Returns:
// - compact duration string such as 350ms, 2.22s, or 1m24s
func FormatDurationCompact(d time.Duration) string {
	if d < time.Second {
		return fmt.Sprintf("%dms", d.Milliseconds())
	}

	if d < time.Minute {
		return fmt.Sprintf("%.2fs", d.Seconds())
	}

	minutes := int(d / time.Minute)
	seconds := int((d % time.Minute) / time.Second)

	return fmt.Sprintf("%dm%02ds", minutes, seconds)
}
