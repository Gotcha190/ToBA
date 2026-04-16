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

	c.Database = DefaultDatabaseName

	return nil
}
