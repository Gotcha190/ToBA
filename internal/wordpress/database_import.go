package wordpress

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/gotcha190/toba/internal/create"
)

// ImportDatabase runs `lando db-import` for sqlPath, converting absolute paths
// to project-relative paths when needed.
//
// Parameters:
// - runner: command runner used to launch the import command
// - projectDir: local project root
// - sqlPath: SQL file path to import
//
// Returns:
// - an error when path normalization or the import command fails
//
// Side effects:
// - runs `lando db-import` in the project directory
func ImportDatabase(runner create.CommandRunner, projectDir string, sqlPath string) error {
	importPath := sqlPath
	if filepath.IsAbs(sqlPath) {
		relative, err := filepath.Rel(projectDir, sqlPath)
		if err != nil {
			return err
		}
		importPath = relative
	}

	if err := runner.Run(projectDir, "lando", "db-import", importPath); err != nil {
		return fmt.Errorf("database import failed: %w", err)
	}

	return nil
}

// SearchReplace rewrites WordPress database URLs from sourceURL to targetURL.
//
// Parameters:
// - runner: command runner used to launch WP-CLI
// - projectDir: local project root
// - sourceURL: original site URL present in the imported database
// - targetURL: replacement local site URL
//
// Returns:
// - an error when the search-replace command fails
//
// Side effects:
// - runs `lando wp search-replace` in the project directory
func SearchReplace(runner create.CommandRunner, projectDir string, sourceURL string, targetURL string) error {
	return replaceInDatabase(runner, projectDir, sourceURL, targetURL, "search-replace failed")
}

// SetConfigTablePrefix updates the wp-config.php table prefix assignment.
//
// Parameters:
// - wpConfigPath: path to the WordPress config file to update
// - prefix: table prefix value to write
//
// Returns:
// - an error when the prefix is invalid or the config file cannot be updated
//
// Side effects:
// - writes wpConfigPath when the table prefix assignment is found
func SetConfigTablePrefix(wpConfigPath string, prefix string) error {
	prefix, err := normalizeTablePrefix(prefix)
	if err != nil {
		return err
	}

	content, err := os.ReadFile(wpConfigPath)
	if err != nil {
		return err
	}

	info, err := os.Stat(wpConfigPath)
	if err != nil {
		return err
	}

	re := regexp.MustCompile(`(?m)^\$table_prefix\s*=\s*'[^']*';`)
	updated := re.ReplaceAllLiteralString(string(content), "$table_prefix = '"+prefix+"';")
	if updated == string(content) {
		return fmt.Errorf("table prefix assignment not found in %s", wpConfigPath)
	}

	return os.WriteFile(wpConfigPath, []byte(updated), info.Mode().Perm())
}

// replaceInDatabase runs a WP-CLI search-replace command for local database
// content.
//
// Parameters:
// - runner: command runner used to launch WP-CLI
// - projectDir: local project root
// - search: source value to replace
// - replace: replacement value
// - errPrefix: context prefix for returned errors
//
// Returns:
// - an error when the search-replace command fails
//
// Side effects:
// - runs `lando wp search-replace`
func replaceInDatabase(runner create.CommandRunner, projectDir string, search string, replace string, errPrefix string) error {
	if err := runner.Run(
		projectDir,
		"lando",
		"wp",
		"search-replace",
		search,
		replace,
		"--all-tables-with-prefix",
		"--skip-columns=guid",
	); err != nil {
		return fmt.Errorf("%s: %w", errPrefix, err)
	}

	return nil
}

// LocalHTTPSURL normalizes a domain or URL string into an HTTPS site URL.
//
// Parameters:
// - domain: raw domain or URL string
//
// Returns:
// - the normalized HTTPS URL
// - an error when the supplied value cannot be interpreted as a valid host
func LocalHTTPSURL(domain string) (string, error) {
	trimmed := strings.TrimSpace(domain)
	if trimmed == "" {
		return "", fmt.Errorf("domain cannot be empty")
	}

	if strings.HasPrefix(trimmed, "http://") || strings.HasPrefix(trimmed, "https://") {
		parsed, err := url.Parse(trimmed)
		if err != nil {
			return "", err
		}
		if parsed.Host == "" {
			return "", fmt.Errorf("invalid domain: %s", domain)
		}

		return "https://" + parsed.Host, nil
	}

	return "https://" + trimmed, nil
}
