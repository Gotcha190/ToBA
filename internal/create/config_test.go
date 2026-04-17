package create

import "testing"

func TestProjectConfigNormalizeLowercasesNameAndDomain(t *testing.T) {
	config := ProjectConfig{
		Name: " Dupa_Test ",
	}

	if err := config.Normalize(); err != nil {
		t.Fatalf("Normalize returned error: %v", err)
	}

	if config.Name != "dupa_test" {
		t.Fatalf("expected lowercase project name, got %q", config.Name)
	}
	if config.Domain != "dupa-test.lndo.site" {
		t.Fatalf("expected lowercase domain, got %q", config.Domain)
	}
}

func TestProjectConfigNormalizeRejectsSpacesInName(t *testing.T) {
	config := ProjectConfig{Name: "demo project"}

	err := config.Normalize()
	if err == nil {
		t.Fatal("expected validation error")
	}
	if err.Error() != `project name cannot contain spaces: "demo project"` {
		t.Fatalf("unexpected error: %v", err)
	}
}
