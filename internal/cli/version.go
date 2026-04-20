package cli

import (
	"fmt"
	"io"
	"os"
	"regexp"
	"runtime/debug"
	"strings"
)

const baseVersion = "1.0.0"
const devSuffix = "dev"

var releaseVersion string
var readBuildInfo = debug.ReadBuildInfo
var pseudoVersionPattern = regexp.MustCompile(`^v\d+\.\d+\.\d+-(?:0\.)?\d{14}-[a-f0-9]{12,}(?:\+dirty)?$`)
var taggedVersionPattern = regexp.MustCompile(`^v\d+\.\d+\.\d+(?:\+dirty)?$`)

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
	runVersionWithWriter(os.Stdout)
}

func runVersionWithWriter(output io.Writer) {
	fmt.Fprintf(output, "toba version: %s\n", resolvedVersion())
}

func resolvedVersion() string {
	if version := normalizeVersion(releaseVersion); version != "" {
		return version
	}

	if info, ok := readBuildInfo(); ok && info != nil {
		if version := normalizeVersion(info.Main.Version); version != "" && !shouldTreatBuildInfoVersionAsDev(info) {
			return version
		}
	}

	return baseVersion + " " + devSuffix
}

func normalizeVersion(raw string) string {
	version := strings.TrimSpace(raw)
	switch version {
	case "", "(devel)":
		return ""
	}

	if strings.HasPrefix(version, "v") && len(version) > 1 {
		next := version[1]
		if next >= '0' && next <= '9' {
			return version[1:]
		}
	}

	return version
}

func shouldTreatBuildInfoVersionAsDev(info *debug.BuildInfo) bool {
	version := strings.TrimSpace(info.Main.Version)
	if !hasVCSMarker(info) {
		return false
	}

	if pseudoVersionPattern.MatchString(version) {
		return true
	}

	return taggedVersionPattern.MatchString(version)
}

func hasVCSMarker(info *debug.BuildInfo) bool {
	for _, setting := range info.Settings {
		if setting.Key == "vcs" && setting.Value != "" {
			return true
		}
	}

	return false
}
