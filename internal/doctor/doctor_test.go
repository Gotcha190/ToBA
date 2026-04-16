package doctor

import "testing"

func TestFullWorkflowChecksIncludeSSHAndZipTools(t *testing.T) {
	checks := FullWorkflowChecks()

	required := map[string]bool{
		"ssh": false,
		"scp": false,
		"zip": false,
	}

	for _, check := range checks {
		if _, ok := required[check.Binary]; ok {
			required[check.Binary] = true
		}
	}

	for binary, found := range required {
		if !found {
			t.Fatalf("expected %s in FullWorkflowChecks", binary)
		}
	}
}
