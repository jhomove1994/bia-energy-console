package engine

import (
	"os/exec"
	"strings"
	"testing"
)

// TestEngineImportsOnlyDomainAndStdlib keeps the engine pure: every
// non-standard-library dependency of the package must be the package itself or
// energy/internal/domain.
func TestEngineImportsOnlyDomainAndStdlib(t *testing.T) {
	out, err := exec.Command("go", "list", "-deps",
		"-f", "{{if not .Standard}}{{.ImportPath}}{{end}}", ".").CombinedOutput()
	if err != nil {
		t.Fatalf("go list failed: %v\n%s", err, out)
	}
	allowed := map[string]bool{
		"energy/internal/engine": true,
		"energy/internal/domain": true,
	}
	for _, dep := range strings.Fields(string(out)) {
		if !allowed[dep] {
			t.Errorf("engine depends on %q; only energy/internal/domain and the standard library are allowed", dep)
		}
	}
}
