package cli

import (
	"runtime/debug"
	"strings"
	"testing"
)

func TestResolvedVersionPrefersReleaseVersion(t *testing.T) {
	originalReleaseVersion := releaseVersion
	originalReadBuildInfo := readBuildInfo
	releaseVersion = "1.0.0"
	readBuildInfo = func() (*debug.BuildInfo, bool) {
		return &debug.BuildInfo{Main: debug.Module{Version: "v9.9.9"}}, true
	}
	t.Cleanup(func() {
		releaseVersion = originalReleaseVersion
		readBuildInfo = originalReadBuildInfo
	})

	if got := resolvedVersion(); got != "1.0.0" {
		t.Fatalf("expected release version, got %q", got)
	}
}

func TestResolvedVersionUsesBuildInfoWhenReleaseVersionIsMissing(t *testing.T) {
	originalReleaseVersion := releaseVersion
	originalReadBuildInfo := readBuildInfo
	releaseVersion = ""
	readBuildInfo = func() (*debug.BuildInfo, bool) {
		return &debug.BuildInfo{Main: debug.Module{Version: "v1.2.3"}}, true
	}
	t.Cleanup(func() {
		releaseVersion = originalReleaseVersion
		readBuildInfo = originalReadBuildInfo
	})

	if got := resolvedVersion(); got != "1.2.3" {
		t.Fatalf("expected build info version, got %q", got)
	}
}

func TestResolvedVersionFallsBackToBaseVersionDev(t *testing.T) {
	originalReleaseVersion := releaseVersion
	originalReadBuildInfo := readBuildInfo
	releaseVersion = ""
	readBuildInfo = func() (*debug.BuildInfo, bool) {
		return &debug.BuildInfo{Main: debug.Module{Version: "(devel)"}}, true
	}
	t.Cleanup(func() {
		releaseVersion = originalReleaseVersion
		readBuildInfo = originalReadBuildInfo
	})

	if got := resolvedVersion(); got != "1.0.0 dev" {
		t.Fatalf("expected dev fallback, got %q", got)
	}
}

func TestResolvedVersionTreatsLocalPseudoVersionAsDev(t *testing.T) {
	originalReleaseVersion := releaseVersion
	originalReadBuildInfo := readBuildInfo
	releaseVersion = ""
	readBuildInfo = func() (*debug.BuildInfo, bool) {
		return &debug.BuildInfo{
			Main: debug.Module{Version: "v0.0.0-20260420100315-3ff11fb608fd"},
			Settings: []debug.BuildSetting{
				{Key: "vcs", Value: "git"},
				{Key: "vcs.revision", Value: "3ff11fb608fd084747bddac1ab056eb7ba77dab8"},
			},
		}, true
	}
	t.Cleanup(func() {
		releaseVersion = originalReleaseVersion
		readBuildInfo = originalReadBuildInfo
	})

	if got := resolvedVersion(); got != "1.0.0 dev" {
		t.Fatalf("expected local pseudo-version to fall back to dev, got %q", got)
	}
}

func TestResolvedVersionTreatsDirtyLocalPseudoVersionAsDev(t *testing.T) {
	originalReleaseVersion := releaseVersion
	originalReadBuildInfo := readBuildInfo
	releaseVersion = ""
	readBuildInfo = func() (*debug.BuildInfo, bool) {
		return &debug.BuildInfo{
			Main: debug.Module{Version: "v0.0.0-20260420100315-3ff11fb608fd+dirty"},
			Settings: []debug.BuildSetting{
				{Key: "vcs", Value: "git"},
				{Key: "vcs.modified", Value: "true"},
			},
		}, true
	}
	t.Cleanup(func() {
		releaseVersion = originalReleaseVersion
		readBuildInfo = originalReadBuildInfo
	})

	if got := resolvedVersion(); got != "1.0.0 dev" {
		t.Fatalf("expected dirty local pseudo-version to fall back to dev, got %q", got)
	}
}

func TestResolvedVersionKeepsPseudoVersionWithoutVCSMarker(t *testing.T) {
	originalReleaseVersion := releaseVersion
	originalReadBuildInfo := readBuildInfo
	releaseVersion = ""
	readBuildInfo = func() (*debug.BuildInfo, bool) {
		return &debug.BuildInfo{
			Main: debug.Module{Version: "v0.0.0-20260420100315-3ff11fb608fd"},
		}, true
	}
	t.Cleanup(func() {
		releaseVersion = originalReleaseVersion
		readBuildInfo = originalReadBuildInfo
	})

	if got := resolvedVersion(); got != "0.0.0-20260420100315-3ff11fb608fd" {
		t.Fatalf("expected pseudo-version without VCS marker to be preserved, got %q", got)
	}
}

func TestRunVersionWritesResolvedVersion(t *testing.T) {
	originalReleaseVersion := releaseVersion
	originalReadBuildInfo := readBuildInfo
	releaseVersion = ""
	readBuildInfo = func() (*debug.BuildInfo, bool) {
		return nil, false
	}
	t.Cleanup(func() {
		releaseVersion = originalReleaseVersion
		readBuildInfo = originalReadBuildInfo
	})

	var output strings.Builder
	runVersionWithWriter(&output)

	if output.String() != "toba version: 1.0.0 dev\n" {
		t.Fatalf("unexpected version output: %q", output.String())
	}
}
