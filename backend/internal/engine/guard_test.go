package engine

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// TestEngineHasNoMeterLiterals keeps dataset knowledge out of the engine: no
// non-test source file, comments included, may name a meter of the demo dataset
// or one of its dates.
func TestEngineHasNoMeterLiterals(t *testing.T) {
	forbidden := []*regexp.Regexp{
		regexp.MustCompile(`M-\d+`),
		regexp.MustCompile(`2026-09-`),
	}

	files, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatal(err)
	}
	scanned := 0
	for _, f := range files {
		if strings.HasSuffix(f, "_test.go") {
			continue
		}
		src, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		scanned++
		for i, line := range strings.Split(string(src), "\n") {
			for _, re := range forbidden {
				if m := re.FindString(line); m != "" {
					t.Errorf("%s:%d contains %q: the engine must not mention dataset meters or dates", f, i+1, m)
				}
			}
		}
	}
	if scanned < 10 {
		t.Fatalf("scanned only %d engine source files; the guard is not looking at the package", scanned)
	}
}
