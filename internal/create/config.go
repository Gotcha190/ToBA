package create

import (
	"fmt"
	"strings"
	"unicode"
)

const DefaultPHPVersion = "8.4"

type ProjectConfig struct {
	Name       string
	PHPVersion string
	Domain     string
	Database   string
	DryRun     bool
}

func (c *ProjectConfig) Normalize() error {
	c.Name = strings.TrimSpace(c.Name)
	if c.Name == "" {
		return fmt.Errorf("project name cannot be empty")
	}

	if strings.TrimSpace(c.PHPVersion) == "" {
		c.PHPVersion = DefaultPHPVersion
	}

	if strings.TrimSpace(c.Domain) == "" {
		c.Domain = fmt.Sprintf("%s.lndo.site", c.Name)
	}

	if strings.TrimSpace(c.Database) == "" {
		c.Database = sanitizeDatabaseName(c.Name)
	}

	return nil
}

func sanitizeDatabaseName(name string) string {
	var builder strings.Builder
	lastUnderscore := false

	for _, r := range strings.ToLower(name) {
		switch {
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			builder.WriteRune(r)
			lastUnderscore = false
		case !lastUnderscore:
			builder.WriteByte('_')
			lastUnderscore = true
		}
	}

	result := strings.Trim(builder.String(), "_")
	if result == "" {
		return "wordpress"
	}

	return result
}
