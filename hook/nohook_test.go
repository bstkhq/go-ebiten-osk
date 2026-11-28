//go:build ebitenosk_nohook

package hook

import (
	"errors"
	"testing"
)

func TestCompiledOutHookReportsWhy(t *testing.T) {
	h, err := New()
	if Supported || h != nil || !errors.Is(err, errDisabled) {
		t.Errorf("New = %v, %v and Supported = %t", h, err, Supported)
	}
}
