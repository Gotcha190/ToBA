package wordpress

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode"

	"github.com/gotcha190/toba/internal/create"
)

// Install downloads WordPress, creates wp-config.php, and performs the initial
// site installation in the local Lando project.
//
// Parameters:
// - runner: command runner used to launch Lando-backed WP-CLI commands
// - projectDir: local project root
// - config: normalized project configuration used for the installation values
//
// Returns:
// - an error when any WordPress bootstrap command fails
//
// Side effects:
// - downloads WordPress core
// - creates wp-config.php
// - performs the initial WordPress install
func Install(runner create.CommandRunner, projectDir string, config create.ProjectConfig) error {
	setupCommands := []string{
		"wp core download --locale=" + shellQuote(defaultLocale),
		"wp config create " +
			"--dbname=" + shellQuote(defaultDBName) + " " +
			"--dbuser=" + shellQuote(defaultDBUser) + " " +
			"--dbpass=" + shellQuote(defaultDBPassword) + " " +
			"--dbhost=" + shellQuote(defaultDBHost) + " " +
			"--dbcharset=" + shellQuote(defaultDBCharset),
	}
	if err := runLandoWPBatch(runner, projectDir, setupCommands...); err != nil {
		return fmt.Errorf("wordpress bootstrap failed: %w", err)
	}

	if err := includeProjectConfigIfPresent(projectDir); err != nil {
		return fmt.Errorf("wp-config project config include failed: %w", err)
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

// ProjectTitle converts a project slug into a human-readable WordPress site
// title.
//
// Parameters:
// - name: project slug or identifier
//
// Returns:
// - the formatted site title
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

// includeProjectConfigIfPresent injects the project config include when a
// root config.php file exists.
//
// Parameters:
// - projectDir: local project root
//
// Returns:
// - an error when config detection or wp-config.php update fails
//
// Side effects:
// - may update app/wp-config.php
func includeProjectConfigIfPresent(projectDir string) error {
	projectConfigPath := filepath.Join(projectDir, "config.php")
	if _, err := os.Stat(projectConfigPath); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	return setProjectConfigInclude(filepath.Join(projectDir, "app", "wp-config.php"))
}

// setProjectConfigInclude inserts the project config include block into
// wp-config.php.
//
// Parameters:
// - wpConfigPath: path to the WordPress config file to update
//
// Returns:
// - an error when the file cannot be read, parsed, or written
//
// Side effects:
// - writes wpConfigPath when the include block is missing
func setProjectConfigInclude(wpConfigPath string) error {
	content, err := os.ReadFile(wpConfigPath)
	if err != nil {
		return err
	}

	info, err := os.Stat(wpConfigPath)
	if err != nil {
		return err
	}

	includeBlock := "" +
		"if ( file_exists( dirname( __DIR__ ) . '/config.php' ) ) {\n" +
		"\trequire_once dirname( __DIR__ ) . '/config.php';\n" +
		"}\n\n"
	if strings.Contains(string(content), "dirname( __DIR__ ) . '/config.php'") {
		return nil
	}

	marker := "/* Add any custom values between this line and the \"stop editing\" line. */"
	index := strings.Index(string(content), marker)
	if index == -1 {
		return fmt.Errorf("custom values marker not found in %s", wpConfigPath)
	}
	index += len(marker)

	updated := string(content[:index]) + "\n\n" + includeBlock + string(content[index:])
	return os.WriteFile(wpConfigPath, []byte(updated), info.Mode().Perm())
}

// runLandoWPBatch runs one or more WP-CLI commands inside the Lando appserver.
//
// Parameters:
// - runner: command runner used to launch Lando
// - projectDir: local project root
// - commands: shell command fragments to run in sequence
//
// Returns:
// - an error when the Lando shell command fails
//
// Side effects:
// - runs `lando ssh -s appserver -c ...`
func runLandoWPBatch(runner create.CommandRunner, projectDir string, commands ...string) error {
	if len(commands) == 0 {
		return nil
	}

	script := "cd /app && " + strings.Join(commands, " && ")
	return runner.Run(projectDir, "lando", "ssh", "-s", "appserver", "-c", script)
}

// shellQuote quotes value for POSIX shell command construction.
//
// Parameters:
// - value: raw string to quote
//
// Returns:
// - the shell-quoted string
func shellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", `'\''`) + "'"
}
