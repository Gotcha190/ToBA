package wordpress

import (
	"fmt"
	"strings"

	"github.com/gotcha190/toba/internal/create"
)

// ResetAdminPassword resets the default local WordPress admin password, or
// creates the default admin account when the imported database does not
// contain it.
//
// Parameters:
// - runner: command runner used to launch WP-CLI
// - projectDir: local project root
//
// Returns:
// - an error when user lookup, creation, or password reset fails
//
// Side effects:
// - runs `lando wp eval ...` to create or update the default admin account
func ResetAdminPassword(runner create.CommandRunner, projectDir string) error {
	script := "" +
		"$user = get_user_by('login', " + phpSingleQuote(defaultAdminUser) + ");" +
		"if ($user) {" +
		"$result = wp_update_user(array('ID' => $user->ID, 'user_pass' => " + phpSingleQuote(defaultAdminPassword) + "));" +
		"if (is_wp_error($result)) {fwrite(STDERR, $result->get_error_message() . PHP_EOL); exit(1);}" +
		"exit(0);" +
		"}" +
		"$user_id = wp_create_user(" + phpSingleQuote(defaultAdminUser) + ", " + phpSingleQuote(defaultAdminPassword) + ", " + phpSingleQuote(defaultAdminEmail) + ");" +
		"if (is_wp_error($user_id)) {fwrite(STDERR, $user_id->get_error_message() . PHP_EOL); exit(1);}" +
		"$result = wp_update_user(array('ID' => $user_id, 'display_name' => " + phpSingleQuote(defaultAdminUser) + ", 'role' => 'administrator'));" +
		"if (is_wp_error($result)) {fwrite(STDERR, $result->get_error_message() . PHP_EOL); exit(1);}"
	_, err := runner.CaptureOutput(
		projectDir,
		"lando",
		"wp",
		"eval",
		script,
	)
	if err != nil {
		return fmt.Errorf("admin password reset failed: %w", err)
	}

	return nil
}

// ActivateTheme activates themeName in the local WordPress installation.
//
// Parameters:
// - runner: command runner used to launch WP-CLI
// - projectDir: local project root
// - themeName: WordPress theme slug to activate
//
// Returns:
// - an error when the theme activation command fails
//
// Side effects:
// - runs `lando wp theme activate ...`
func ActivateTheme(runner create.CommandRunner, projectDir string, themeName string) error {
	if err := runner.Run(projectDir, "lando", "wp", "theme", "activate", themeName); err != nil {
		return fmt.Errorf("theme activation failed: %w", err)
	}

	return nil
}

// DetectImportedThemeSlug reads the active theme slug from the imported
// database by checking the stylesheet and template options.
//
// Parameters:
// - runner: command runner used to query WP-CLI options
// - projectDir: local project root
//
// Returns:
// - the detected theme slug
// - an error when both option lookups fail or return empty values
//
// Side effects:
// - runs `lando wp eval ...` to read the stylesheet or template option
func DetectImportedThemeSlug(runner create.CommandRunner, projectDir string) (string, error) {
	value, err := runner.CaptureOutput(
		projectDir,
		"lando",
		"wp",
		"eval",
		"echo get_option('stylesheet') ?: get_option('template');",
	)
	if err != nil {
		return "", fmt.Errorf("theme detection failed: %w", err)
	}

	slug := strings.TrimSpace(value)
	if slug != "" {
		return slug, nil
	}

	return "", fmt.Errorf("theme detection failed: imported database does not expose stylesheet/template; make sure the restored database points to the imported theme and activate the theme manually if needed")
}

// FlushRewriteRules runs the hard rewrite flush command in the local project.
//
// Parameters:
// - runner: command runner used to launch WP-CLI
// - projectDir: local project root
//
// Returns:
// - an error when the rewrite flush command fails
//
// Side effects:
// - runs `lando wp rewrite flush --hard`
func FlushRewriteRules(runner create.CommandRunner, projectDir string) error {
	if err := runner.Run(projectDir, "lando", "wp", "rewrite", "flush", "--hard"); err != nil {
		return fmt.Errorf("rewrite flush failed: %w", err)
	}

	return nil
}

// RefreshThemeCaches runs the Acorn commands used to rebuild theme caches.
//
// Parameters:
// - runner: command runner used to launch Acorn commands
// - projectDir: local project root
//
// Returns:
// - an error when any cache maintenance command fails
//
// Side effects:
// - runs supported `wp acorn ...` commands in one Lando appserver shell
func RefreshThemeCaches(runner create.CommandRunner, projectDir string) error {
	acornList, err := runner.CaptureOutput(projectDir, "lando", "wp", "acorn", "list")
	if err != nil {
		return fmt.Errorf("theme cache command discovery failed: %w", err)
	}

	var commands [][]string
	switch {
	case acornCommandAvailable(acornList, "cache:clear"):
		commands = append(commands, []string{"wp", "acorn", "optimize"})
		commands = append(commands, []string{"wp", "acorn", "cache:clear"})
	case acornCommandAvailable(acornList, "optimize:clear"):
		commands = append(commands, []string{"wp", "acorn", "optimize:clear"})
	case acornCommandAvailable(acornList, "config:clear"):
		commands = append(commands, []string{"wp", "acorn", "config:clear"})
	default:
		return fmt.Errorf("theme cache refresh failed: no supported Acorn cache clear command found")
	}

	if acornCommandAvailable(acornList, "acf:cache") {
		commands = append(commands, []string{"wp", "acorn", "acf:cache"})
	}

	commandScripts := make([]string, 0, len(commands))
	for _, args := range commands {
		commandName := args[len(args)-1]
		if acornCommandAvailable(acornList, commandName) {
			commandScripts = append(commandScripts, strings.Join(args, " "))
		}
	}

	if err := runLandoWPBatch(runner, projectDir, commandScripts...); err != nil {
		if len(commandScripts) == 1 {
			return fmt.Errorf("theme cache refresh failed for %s: %w", commandScripts[0], err)
		}
		return fmt.Errorf("theme cache refresh failed for batched acorn commands: %w", err)
	}

	return nil
}

// phpSingleQuote quotes value for PHP single-quoted string literals.
//
// Parameters:
// - value: raw string to quote
//
// Returns:
// - the PHP single-quoted string literal
func phpSingleQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", `\'`) + "'"
}

// acornCommandAvailable reports whether an Acorn command appears in command
// list output.
//
// Parameters:
// - listOutput: output from `wp acorn list`
// - command: Acorn command name to search for
//
// Returns:
// - true when command is present
func acornCommandAvailable(listOutput string, command string) bool {
	for _, line := range strings.Split(listOutput, "\n") {
		fields := strings.Fields(strings.TrimSpace(line))
		if len(fields) > 0 && fields[0] == command {
			return true
		}
	}

	return false
}
