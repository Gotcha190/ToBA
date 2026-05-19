package sourcedata

import (
	"fmt"
	"net/url"
	"strings"
)

// remotePreparationScript builds the shell script that prepares remote starter artifacts.
//
// Returns:
// - the shell script executed on the SSH host
func remotePreparationScript(remoteWordPressRoot string, remoteDatabase string, remotePlugins string, remoteUploads string, remoteSourceURL string, includeUploads bool) string {
	commands := []string{
		"set -eu",
		"if [ ! -d " + shellQuote(remoteWordPressRoot) + " ]; then printf '%s\\n' " + shellQuote("__TOBA_REMOTE_ROOT_MISSING__") + "; exit 42; fi",
		"cleanup_on_error() { status=$?; if [ \"$status\" -ne 0 ]; then rm -f " + shellQuote(remoteDatabase) + " " + shellQuote(remotePlugins) + " " + shellQuote(remoteUploads) + " " + shellQuote(remoteSourceURL) + "; fi; exit \"$status\"; }",
		"cleanup_on_signal() { rm -f " + shellQuote(remoteDatabase) + " " + shellQuote(remotePlugins) + " " + shellQuote(remoteUploads) + " " + shellQuote(remoteSourceURL) + "; exit 130; }",
		"trap cleanup_on_error EXIT",
		"trap cleanup_on_signal HUP INT TERM",
		"cd " + shellQuote(remoteWordPressRoot),
		"wp84 option get home > " + shellQuote(pathBase(remoteSourceURL)) + " & pid_source=$!",
		"wp84 db export " + shellQuote(pathBase(remoteDatabase)) + " >/dev/null & pid_db=$!",
		"(cd wp-content && zip -r -q ../" + shellQuote(pathBase(remotePlugins)) + " plugins) & pid_plugins=$!",
	}
	if includeUploads {
		commands = append(commands, "(cd wp-content && zip -r -q -0 ../"+shellQuote(pathBase(remoteUploads))+" . -i "+shellQuote("uploads/*")+") & pid_uploads=$!")
	}
	commands = append(commands,
		"wait \"$pid_source\"",
		"wait \"$pid_db\"",
		"wait \"$pid_plugins\"",
	)
	if includeUploads {
		commands = append(commands, "wait \"$pid_uploads\"")
	}
	commands = append(commands, "cat "+shellQuote(pathBase(remoteSourceURL)))

	return strings.Join(commands, "; ")
}

// normalizeSourceURL validates the captured remote site URL and returns a normalized string form.
//
// Returns:
// - a normalized URL string
// - an error when the URL is missing a scheme or host
func normalizeSourceURL(raw string) (string, error) {
	lines := strings.Split(raw, "\n")
	for i := len(lines) - 1; i >= 0; i-- {
		candidate := strings.TrimSpace(lines[i])
		if candidate == "" {
			continue
		}

		parsed, err := url.Parse(candidate)
		if err != nil {
			continue
		}
		if parsed.Scheme == "" || parsed.Host == "" {
			continue
		}

		return parsed.String(), nil
	}

	return "", fmt.Errorf("invalid remote WordPress home URL: %s", raw)
}
