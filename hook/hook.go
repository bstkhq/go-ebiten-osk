package hook

import (
	"errors"
	"fmt"
	"runtime/debug"
	"strconv"
	"strings"

	osk "github.com/bstkhq/go-ebiten-osk"
	"github.com/hajimehoshi/ebiten/v2"
)

const engine = "github.com/hajimehoshi/ebiten/v2"

var errDisabled = errors.New("input hook compiled out with ebitenosk_nohook")

type engineBackend struct {
	verify func() error
	apply  func(*Hook)
}

// Hook writes the current tick's keyboard updates into Ebitengine. Use one
// Hook for the game and share its option with all keyboards that feed it.
type Hook struct {
	pending  frame
	applied  int64
	heldFrom [ebiten.KeyMax + 1]inputTime
}

// New checks the linked Ebitengine version and probes its input snapshot.
// Call it during game setup or at the start of the first Update, before any
// input reads. If it returns an error, create keyboards with the input copy.
func New() (*Hook, error) {
	if !Supported {
		return nil, errDisabled
	}
	info, _ := debug.ReadBuildInfo()
	return newHook(engineVersion(info))
}

func engineVersion(info *debug.BuildInfo) string {
	if info != nil {
		for _, dep := range info.Deps {
			if dep.Path == engine {
				if dep.Replace != nil {
					return "replacement module (" + dep.Replace.Version + ")"
				}
				return dep.Version
			}
		}
	}
	return "unknown"
}

func newHook(version string) (*Hook, error) {
	if !supportedVersion(version) {
		return nil, fmt.Errorf("input hook supports Ebitengine v2.9.x releases, found %q", version)
	}
	if err := backend.verify(); err != nil {
		return nil, fmt.Errorf("input hook self-check for Ebitengine %s: %w", version, err)
	}
	return &Hook{applied: -1, pending: frame{tick: -1}}, nil
}

func supportedVersion(version string) bool {
	patch, ok := strings.CutPrefix(version, "v2.9.")
	if !ok {
		return false
	}
	n, err := strconv.ParseUint(patch, 10, 64)
	return err == nil && strconv.FormatUint(n, 10) == patch
}

// Option routes a keyboard to the hook. Pass it to [osk.New].
func (h *Hook) Option() osk.Option {
	return osk.WithInputWriter(h.capture)
}

// Update writes keys and characters for the current tick. Call it inside
// the game's Update, after updating all keyboards attached to this hook and
// before reading keyboard input or updating keyboards that feed the input
// copy. Keep calling it when a keyboard is dropped so its keys are released.
func (h *Hook) Update() {
	tick := ebiten.Tick()
	if tick == h.applied {
		return
	}
	if tick != h.applied+1 {
		clear(h.heldFrom[:])
	}
	h.applied = tick
	h.pending.start(tick)
	backend.apply(h)
}
