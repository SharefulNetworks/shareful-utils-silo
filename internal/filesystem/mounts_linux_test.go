//go:build linux

package filesystem

import "testing"

func TestNewDefaultMountHandlerIncludesCoreRuntimeMounts(t *testing.T) {
	mh := NewDefaultMountHandler()
	if mh == nil {
		t.Fatal("expected non-nil mount handler")
	}

	want := map[string]bool{
		"proc":     true,
		"tmpfs":    true,
		"devtmpfs": true,
	}

	seen := map[string]bool{}
	for _, spec := range mh.mounts {
		seen[spec.Type] = true
	}

	for typ := range want {
		if !seen[typ] {
			t.Fatalf("expected default mount handler to include type %q, got %#v", typ, seen)
		}
	}
}
