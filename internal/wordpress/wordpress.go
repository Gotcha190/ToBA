package wordpress

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
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
	DefaultTablePrefix   = "wp_"
	metadataHeaderLimit  = 64 * 1024
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

// BackupSourceURL reads the original site URL from SQL backup metadata.
//
// Parameters:
// - sqlPath: SQL dump path to inspect
//
// Returns:
// - the validated source URL found in the backup comments
// - an error when the file cannot be read or the URL metadata is missing/invalid
func BackupSourceURL(sqlPath string) (string, error) {
	file, err := os.Open(sqlPath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	var backupURL string
	parseLine := func(line string) error {
		line = strings.TrimSpace(line)
		switch {
		case strings.HasPrefix(line, "# Home URL:"):
			value, err := validateURL(strings.TrimSpace(strings.TrimPrefix(line, "# Home URL:")))
			if err != nil {
				return err
			}
			backupURL = value
			return errStopReading
		case backupURL == "" && strings.HasPrefix(line, "# Backup of:"):
			backupURL = strings.TrimSpace(strings.TrimPrefix(line, "# Backup of:"))
		}
		return nil
	}

	err = scanMetadataHeader(file, parseLine)
	if err != nil && !errors.Is(err, errStopReading) {
		return "", err
	}
	if backupURL != "" {
		return validateURL(backupURL)
	}

	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return "", err
	}
	err = forEachLine(file, parseLine)
	if err != nil && !errors.Is(err, errStopReading) {
		return "", err
	}
	if backupURL == "" {
		return "", fmt.Errorf("backup source URL not found in %s", sqlPath)
	}

	return validateURL(backupURL)
}

// BackupTablePrefix reads the original WordPress table prefix from SQL backup
// metadata, falling back to the default prefix when the dump does not expose
// one.
//
// Parameters:
// - sqlPath: SQL dump path to inspect
//
// Returns:
// - the detected or default table prefix
// - an error when the file cannot be read or metadata is invalid
func BackupTablePrefix(sqlPath string) (string, error) {
	file, err := os.Open(sqlPath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	var tablePrefix string
	parseLine := func(line string) error {
		line = strings.TrimSpace(line)
		switch {
		case strings.HasPrefix(line, "# Table prefix:"):
			value, err := normalizeTablePrefix(strings.TrimSpace(strings.TrimPrefix(line, "# Table prefix:")))
			if err != nil {
				return err
			}
			tablePrefix = value
			return errStopReading
		case strings.HasPrefix(line, "# Table: `"), strings.HasPrefix(line, "CREATE TABLE `"), strings.HasPrefix(line, "DROP TABLE IF EXISTS `"):
			tableName := tableNameFromLine(line)
			if prefix, ok := tablePrefixFromTableName(tableName); ok {
				tablePrefix = prefix
				return errStopReading
			}
		}
		return nil
	}

	err = scanMetadataHeader(file, parseLine)
	if err != nil && !errors.Is(err, errStopReading) {
		return "", err
	}
	if tablePrefix != "" {
		return tablePrefix, nil
	}

	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return "", err
	}
	err = forEachLine(file, parseLine)
	if err != nil && !errors.Is(err, errStopReading) {
		return "", err
	}
	if tablePrefix != "" {
		return tablePrefix, nil
	}

	return DefaultTablePrefix, nil
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

// normalizeTablePrefix trims and validates a WordPress table prefix.
//
// Parameters:
// - prefix: raw table prefix value
//
// Returns:
// - the normalized prefix
// - an error when the prefix is empty or contains invalid characters
func normalizeTablePrefix(prefix string) (string, error) {
	prefix = strings.TrimSpace(prefix)
	if prefix == "" {
		return "", fmt.Errorf("table prefix is empty")
	}
	for _, r := range prefix {
		if r != '_' && (r < '0' || r > '9') && (r < 'A' || r > 'Z') && (r < 'a' || r > 'z') {
			return "", fmt.Errorf("invalid table prefix: %s", prefix)
		}
	}

	return prefix, nil
}

// tableNameFromLine extracts a SQL table name from supported dump lines.
//
// Parameters:
// - line: SQL dump line to inspect
//
// Returns:
// - the table name, or an empty string when none can be parsed
func tableNameFromLine(line string) string {
	for _, marker := range []string{"# Table: `", "CREATE TABLE `", "DROP TABLE IF EXISTS `"} {
		if !strings.HasPrefix(line, marker) {
			continue
		}
		name := strings.TrimPrefix(line, marker)
		end := strings.IndexByte(name, '`')
		if end <= 0 {
			return ""
		}
		return name[:end]
	}

	return ""
}

// tablePrefixFromTableName infers a WordPress table prefix from a table name.
//
// Parameters:
// - tableName: database table name to inspect
//
// Returns:
// - the inferred table prefix
// - true when the table name matches a known WordPress table suffix
func tablePrefixFromTableName(tableName string) (string, bool) {
	for _, suffix := range []string{
		"options",
		"users",
		"usermeta",
		"posts",
		"postmeta",
		"terms",
		"term_taxonomy",
		"term_relationships",
		"termmeta",
		"commentmeta",
		"comments",
		"links",
	} {
		if !strings.HasSuffix(tableName, suffix) || len(tableName) <= len(suffix) {
			continue
		}

		prefix, err := normalizeTablePrefix(strings.TrimSuffix(tableName, suffix))
		if err == nil {
			return prefix, true
		}
	}

	return "", false
}

var errStopReading = errors.New("stop reading")

// scanMetadataHeader scans the bounded SQL dump header for metadata lines.
//
// Parameters:
// - file: SQL dump file positioned at the beginning
// - fn: callback invoked for each line
//
// Returns:
// - an error returned by the reader or callback
//
// Side effects:
// - advances the file read position
func scanMetadataHeader(file *os.File, fn func(line string) error) error {
	return forEachLine(io.LimitReader(file, metadataHeaderLimit), fn)
}

// forEachLine streams lines from reader without imposing scanner token limits.
//
// Parameters:
// - reader: source stream to read
// - fn: callback invoked for each line
//
// Returns:
// - an error returned by the reader or callback
func forEachLine(reader io.Reader, fn func(line string) error) error {
	buf := make([]byte, 0, 64*1024)
	lineBuf := bytes.NewBuffer(buf)
	chunk := make([]byte, 32*1024)

	for {
		n, err := reader.Read(chunk)
		if n > 0 {
			start := 0
			for i := 0; i < n; i++ {
				if chunk[i] != '\n' {
					continue
				}

				lineBuf.Write(chunk[start:i])
				line := strings.TrimSuffix(lineBuf.String(), "\r")
				lineBuf.Reset()
				if callErr := fn(line); callErr != nil {
					return callErr
				}
				start = i + 1
			}
			if start < n {
				lineBuf.Write(chunk[start:n])
			}
		}

		if err != nil {
			if err == io.EOF {
				if lineBuf.Len() == 0 {
					return nil
				}
				return fn(strings.TrimSuffix(lineBuf.String(), "\r"))
			}
			return err
		}
	}
}

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

// validateURL parses raw and rejects values that do not contain both scheme
// and host parts.
//
// Parameters:
// - raw: raw URL string to validate
//
// Returns:
// - the normalized URL string
// - an error when the URL is missing a scheme or host
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
