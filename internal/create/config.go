package create

import (
	"fmt"
	"strings"
)

const DefaultPHPVersion = "8.4"
const DefaultDatabaseName = "wordpress"

type ProjectConfig struct {
	Name        string
	PHPVersion  string
	Domain      string
	Database    string
	StarterRepo string
	SSHTarget   string
	DryRun      bool
}

func (c *ProjectConfig) Normalize() error {
	c.Name = strings.ToLower(strings.TrimSpace(c.Name))
	if c.Name == "" {
		return fmt.Errorf("project name cannot be empty")
	}
	if strings.ContainsAny(c.Name, " \t\r\n") {
		return fmt.Errorf("project name cannot contain spaces: %q", c.Name)
	}
	for _, char := range c.Name {
		if (char >= 'a' && char <= 'z') || (char >= '0' && char <= '9') || char == '-' || char == '_' {
			continue
		}
		return fmt.Errorf("project name can only contain lowercase letters, numbers, hyphens, and underscores: %q", c.Name)
	}

	if strings.TrimSpace(c.PHPVersion) == "" {
		c.PHPVersion = DefaultPHPVersion
	}

	c.Domain = fmt.Sprintf("%s.lndo.site", strings.ReplaceAll(c.Name, "_", "-"))

	c.Database = DefaultDatabaseName

	return nil
}
