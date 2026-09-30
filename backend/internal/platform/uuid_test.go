package platform

import (
	"regexp"
	"testing"
)

var uuidV4 = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)

func TestNewIDIsRFC4122V4(t *testing.T) {
	for i := 0; i < 50; i++ {
		id := NewID()
		if !uuidV4.MatchString(id) {
			t.Fatalf("NewID() = %q, want RFC 4122 v4 format", id)
		}
	}
}

func TestNewIDIsUnique(t *testing.T) {
	seen := make(map[string]struct{}, 1000)
	for i := 0; i < 1000; i++ {
		id := NewID()
		if _, dup := seen[id]; dup {
			t.Fatalf("duplicate id %q after %d calls", id, i)
		}
		seen[id] = struct{}{}
	}
	if len(seen) != 1000 {
		t.Fatalf("got %d distinct ids, want 1000", len(seen))
	}
}

func TestUUIDGeneratorImplementsPort(t *testing.T) {
	var g UUIDGenerator
	if !uuidV4.MatchString(g.NewID()) {
		t.Fatal("UUIDGenerator.NewID must return a v4 uuid")
	}
}
