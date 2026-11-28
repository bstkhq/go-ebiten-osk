package hook

import (
	"runtime/debug"
	"strings"
	"testing"
)

func TestEngineVersionFromBuildInfo(t *testing.T) {
	for _, tc := range []struct {
		name string
		info *debug.BuildInfo
		want string
	}{
		{"missing build info", nil, "unknown"},
		{"missing dependency", &debug.BuildInfo{Deps: []*debug.Module{{Path: "example.org/other"}}}, "unknown"},
		{"linked release", &debug.BuildInfo{Deps: []*debug.Module{{Path: engine, Version: "v2.9.9"}}}, "v2.9.9"},
		{"replaced source", &debug.BuildInfo{Deps: []*debug.Module{{Path: engine, Version: "v2.9.9", Replace: &debug.Module{Version: "v2.10.1"}}}}, "replacement module (v2.10.1)"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := engineVersion(tc.info); got != tc.want {
				t.Errorf("engineVersion = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestUnsupportedVersionsFailBeforeConnectingToEngine(t *testing.T) {
	for _, version := range []string{"unknown", "v2.8.9", "v2.10.1", "v2.9.9-rc.1", "v2.9.01", "v2.9."} {
		got, err := newHook(version)
		if got != nil || err == nil || !strings.Contains(err.Error(), version) {
			t.Errorf("newHook(%q) = %v, %v, want a version error", version, got, err)
		}
	}
}
