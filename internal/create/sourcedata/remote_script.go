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
	cleanupFiles := shellQuote(pathBase(remoteDatabase)) + " " + shellQuote(pathBase(remotePlugins)) + " " + shellQuote(pathBase(remoteUploads)) + " " + shellQuote(pathBase(remoteSourceURL))
	commands := []string{
		"set -eu",
		"if [ ! -d " + shellQuote(remoteWordPressRoot) + " ]; then printf '%s\\n' " + shellQuote("__TOBA_REMOTE_ROOT_MISSING__") + "; exit 42; fi",
		"cd " + shellQuote(remoteWordPressRoot),
		"pids=''",
		"cleanup_on_error() { status=$?; trap - EXIT HUP INT TERM; if [ \"$status\" -ne 0 ]; then for pid in $pids; do kill \"$pid\" 2>/dev/null || :; done; for pid in $pids; do wait \"$pid\" 2>/dev/null || :; done; rm -f " + cleanupFiles + "; fi; exit \"$status\"; }",
		"trap cleanup_on_error EXIT",
		"trap 'exit 130' HUP INT TERM",
		"wp84 option get home > " + shellQuote(pathBase(remoteSourceURL)) + " & pid_source=$!; pids=\"$pids $pid_source\"",
		"wp84 db export " + shellQuote(pathBase(remoteDatabase)) + " >/dev/null & pid_db=$!; pids=\"$pids $pid_db\"",
		"(cd wp-content && exec zip -r -q ../" + shellQuote(pathBase(remotePlugins)) + " plugins) & pid_plugins=$!; pids=\"$pids $pid_plugins\"",
	}
	if includeUploads {
		commands = append(commands, "(cd wp-content && exec zip -r -q -0 ../"+shellQuote(pathBase(remoteUploads))+" . -i "+shellQuote("uploads/*")+") & pid_uploads=$!; pids=\"$pids $pid_uploads\"")
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
