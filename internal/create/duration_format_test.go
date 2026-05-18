package create

import (
	"testing"
	"time"
)

func TestFormatDurationCompact(t *testing.T) {
	tests := []struct {
		name     string
		duration time.Duration
		want     string
	}{
		{name: "milliseconds", duration: 350 * time.Millisecond, want: "350ms"},
		{name: "max milliseconds", duration: 999 * time.Millisecond, want: "999ms"},
		{name: "one second", duration: time.Second, want: "1.00s"},
		{name: "fractional seconds", duration: 2220 * time.Millisecond, want: "2.22s"},
		{name: "many seconds", duration: 59900 * time.Millisecond, want: "59.90s"},
		{name: "one minute", duration: time.Minute, want: "1m00s"},
		{name: "minutes and seconds", duration: 84 * time.Second, want: "1m24s"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := FormatDurationCompact(tt.duration); got != tt.want {
				t.Fatalf("expected %q, got %q", tt.want, got)
			}
		})
	}
}
