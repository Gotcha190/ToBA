package wordpress

import (
	"fmt"
	"strings"
	"unicode"

	"github.com/gotcha190/ToBA/internal/create"
)

const (
	defaultLocale        = "pl_PL"
	defaultDBName        = "wordpress"
	defaultDBUser        = "wordpress"
	defaultDBPassword    = "wordpress"
	defaultDBHost        = "database"
	defaultAdminUser     = "tamago"
	defaultAdminEmail    = "email@email.pl"
	defaultAdminPassword = "tamago"
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
