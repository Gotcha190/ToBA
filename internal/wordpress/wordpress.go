package wordpress

import (
	"bufio"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"unicode"

	"github.com/gotcha190/toba/internal/create"
)

const (
	defaultLocale        = "pl_PL"
	defaultDBName        = "wordpress"
	defaultDBUser        = "wordpress"
	defaultDBPassword    = "wordpress"
	defaultDBHost        = "database"
	defaultDBCharset     = "utf8mb4"
	defaultAdminUser     = "tamago"
	defaultAdminEmail    = "email@email.pl"
	defaultAdminPassword = "tamago"
	defaultAdminID       = "1"
)

func Install(runner create.CommandRunner, projectDir string, config create.ProjectConfig) error {
	if err := runner.Run(projectDir, "lando", "wp", "core", "download", "--locale="+defaultLocale); err != nil {
		return fmt.Errorf("wordpress download failed: %w", err)
	}

	if err := runner.Run(
		projectDir,
		"lando",
		"wp",
		"config",
		"create",
		"--dbname="+defaultDBName,
		"--dbuser="+defaultDBUser,
		"--dbpass="+defaultDBPassword,
		"--dbhost="+defaultDBHost,
		"--dbcharset="+defaultDBCharset,
	); err != nil {
		return fmt.Errorf("wp-config creation failed: %w", err)
	}

	if err := runner.Run(
		projectDir,
		"lando",
		"wp",
		"core",
		"install",
		"--url="+config.Domain,
		"--title="+ProjectTitle(config.Name),
		"--admin_user="+defaultAdminUser,
		"--admin_email="+defaultAdminEmail,
		"--admin_password="+defaultAdminPassword,
	); err != nil {
		return fmt.Errorf("wordpress install failed: %w", err)
	}

	return nil
}

func ProjectTitle(name string) string {
	normalized := strings.NewReplacer("-", " ", "_", " ").Replace(strings.TrimSpace(name))
	words := strings.Fields(normalized)
	if len(words) == 0 {
		return name
	}

	for i, word := range words {
		runes := []rune(strings.ToLower(word))
		runes[0] = unicode.ToUpper(runes[0])
		words[i] = string(runes)
	}

	return strings.Join(words, " ")
}

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

func BackupSourceURL(sqlPath string) (string, error) {
	file, err := os.Open(sqlPath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	var backupURL string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		switch {
		case strings.HasPrefix(line, "# Home URL:"):
			return validateURL(strings.TrimSpace(strings.TrimPrefix(line, "# Home URL:")))
		case backupURL == "" && strings.HasPrefix(line, "# Backup of:"):
			backupURL = strings.TrimSpace(strings.TrimPrefix(line, "# Backup of:"))
		}
	}
	if err := scanner.Err(); err != nil {
		return "", err
	}
	if backupURL == "" {
		return "", fmt.Errorf("backup source URL not found in %s", sqlPath)
	}

	return validateURL(backupURL)
}

func SearchReplace(runner create.CommandRunner, projectDir string, sourceURL string, targetURL string) error {
	if err := runner.Run(
		projectDir,
		"lando",
		"wp",
		"search-replace",
		sourceURL,
		targetURL,
		"--all-tables-with-prefix",
		"--skip-columns=guid",
	); err != nil {
		return fmt.Errorf("search-replace failed: %w", err)
	}

	return nil
}

func ResetAdminPassword(runner create.CommandRunner, projectDir string) error {
	if err := runner.Run(
		projectDir,
		"lando",
		"wp",
		"user",
		"update",
		defaultAdminID,
		"--user_pass="+defaultAdminPassword,
	); err != nil {
		return fmt.Errorf("admin password reset failed: %w", err)
	}

	return nil
}

func ActivateTheme(runner create.CommandRunner, projectDir string, themeName string) error {
	if err := runner.Run(projectDir, "lando", "wp", "theme", "activate", themeName); err != nil {
		return fmt.Errorf("theme activation failed: %w", err)
	}

	return nil
}

func DetectImportedThemeSlug(runner create.CommandRunner, projectDir string) (string, error) {
	for _, option := range []string{"stylesheet", "template"} {
		value, err := runner.CaptureOutput(projectDir, "lando", "wp", "option", "get", option)
		if err != nil {
			return "", fmt.Errorf("theme detection failed for option %s: %w", option, err)
		}

		slug := strings.TrimSpace(value)
		if slug != "" {
			return slug, nil
		}
	}

	return "", fmt.Errorf("theme detection failed: imported database does not expose stylesheet/template; make sure the restored database points to the imported theme and activate the theme manually if needed")
}

func FlushRewriteRules(runner create.CommandRunner, projectDir string) error {
	if err := runner.Run(projectDir, "lando", "wp", "rewrite", "flush", "--hard"); err != nil {
		return fmt.Errorf("rewrite flush failed: %w", err)
	}

	return nil
}

func RefreshThemeCaches(runner create.CommandRunner, projectDir string) error {
	for _, args := range [][]string{
		{"wp", "acorn", "optimize"},
		{"wp", "acorn", "cache:clear"},
		{"wp", "acorn", "acf:cache"},
	} {
		if err := runner.Run(projectDir, "lando", args...); err != nil {
			return fmt.Errorf("theme cache refresh failed for %s: %w", strings.Join(args, " "), err)
		}
	}

	return nil
}

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

func validateURL(raw string) (string, error) {
	parsed, err := url.Parse(raw)
	if err != nil {
		return "", err
	}
	if parsed.Scheme == "" || parsed.Host == "" {
		return "", fmt.Errorf("invalid backup URL: %s", raw)
	}

	return parsed.String(), nil
}
