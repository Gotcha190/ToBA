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

func TestRunChecksReturnsResultPerCheck(t *testing.T) {
	results := RunChecks([]Check{{Name: "Missing", Binary: "binary-that-does-not-exist-toba"}})
	if len(results) != 1 {
		t.Fatalf("expected a single result, got %d", len(results))
	}
	if results[0].Check.Binary != "binary-that-does-not-exist-toba" {
		t.Fatalf("unexpected result: %#v", results[0])
	}
	if results[0].Err == nil {
		t.Fatal("expected missing binary error")
	}
}
