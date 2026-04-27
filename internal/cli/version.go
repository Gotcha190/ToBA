package cli

import (
	"fmt"
	"io"
	"os"
	"regexp"
	"runtime/debug"
	"strings"
)

const baseVersion = "1.2.1"
const devSuffix = "dev"

var releaseVersion string
var readBuildInfo = debug.ReadBuildInfo
var pseudoVersionPattern = regexp.MustCompile(`^v\d+\.\d+\.\d+-(?:0\.)?\d{14}-[a-f0-9]{12,}(?:\+dirty)?$`)
var taggedVersionPattern = regexp.MustCompile(`^v\d+\.\d+\.\d+(?:\+dirty)?$`)

// RunVersion prints the current CLI version string.
//
// Returns:
// - null
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

// runVersionWithWriter writes the resolved CLI version to output.
//
// Parameters:
// - output: destination writer for the version line
//
// Returns:
// - null
//
// Side effects:
// - writes the version followed by a newline
func runVersionWithWriter(output io.Writer) {
	_, _ = fmt.Fprintf(output, "toba version: %s\n", resolvedVersion())
}

// resolvedVersion picks the best available version string for the current
// binary.
//
// Returns:
// - the release version, build-info version, or base development version
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

// normalizeVersion strips supported version prefixes and ignores empty
// development placeholders.
//
// Parameters:
// - raw: version string reported by release metadata or build info
//
// Returns:
// - the normalized version string, or an empty string when raw has no usable version
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

// shouldTreatBuildInfoVersionAsDev reports whether build info describes a
// local development binary.
//
// Parameters:
// - info: build metadata returned by debug.ReadBuildInfo
//
// Returns:
// - true when the binary should display the development version label
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

// hasVCSMarker reports whether build info contains VCS metadata.
//
// Parameters:
// - info: build metadata returned by debug.ReadBuildInfo
//
// Returns:
// - true when a VCS marker is present
func hasVCSMarker(info *debug.BuildInfo) bool {
	for _, setting := range info.Settings {
		if setting.Key == "vcs" && setting.Value != "" {
			return true
		}
	}

	return false
}
