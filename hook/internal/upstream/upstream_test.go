package upstream

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// Adapted from go-ebiten-mcp v0.3.2 (MIT). See ../../LICENSE.

const module = "github.com/hajimehoshi/ebiten/v2"

func TestEngineDeclarations(t *testing.T) {
	dir := moduleDir(t)
	checks := []struct {
		path  string
		names []string
		hash  string
	}{
		{"internal/ui/input.go", []string{"MouseButton", "MouseButton0", "TouchID", "Touch", "InputState"}, "fe89eeee46eb07dc4cc887c8e7fd6082c61ccccb35e466d0514ef1e532041387"},
		{"internal/ui/keys.go", []string{"Key", "KeyA"}, "33fc8af021925bbc845889c91505e65e0201292455016446430ded4bcd4ea2a4"},
		{"internal/ui/ui.go", []string{"Get", "Tick", "incrementTick", "InputTime", "inputTimeSubtickBits"}, "36540282bccdcf27ca3f047b3e0cdf556b10e2107881a33ac266d01d9b34c789"},
		{"internal/inputstate/inputstate.go", []string{"Get", "Update", "inputState"}, "195a5ed67716593d2d629f7b0913789d9183ea3ae875c2aa32bf5cd63aea9bb8"},
	}
	for _, check := range checks {
		t.Run(check.path, func(t *testing.T) {
			src := declarations(t, filepath.Join(dir, check.path), check.names)
			if got := fmt.Sprintf("%x", sha256.Sum256(src)); got != check.hash {
				t.Errorf("Ebitengine declarations changed, got %s, want %s. Reverify the hook before updating the hash.", got, check.hash)
			}
		})
	}
}

func moduleDir(t *testing.T) string {
	t.Helper()
	out, err := exec.Command("go", "list", "-m", "-f", "{{.Dir}}", module).Output()
	if dir := strings.TrimSpace(string(out)); err == nil && dir != "" {
		return dir
	}
	t.Fatalf("find Ebitengine sources: %v (run go mod download %s)", err, module)
	return ""
}

func declarations(t *testing.T, path string, names []string) []byte {
	t.Helper()
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, path, nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	for _, name := range names {
		matched := false
		for _, decl := range file.Decls {
			if !slices.Contains(declarationNames(decl), name) {
				continue
			}
			matched = true
			// Include the whole const group because changing an earlier iota
			// changes the bounds even when the final constant is unchanged.
			if err := format.Node(&out, fset, decl); err != nil {
				t.Fatal(err)
			}
			out.WriteByte('\n')
		}
		if !matched {
			t.Fatalf("%s has no declaration named %s", path, name)
		}
	}
	return out.Bytes()
}

func declarationNames(decl ast.Decl) []string {
	switch d := decl.(type) {
	case *ast.FuncDecl:
		return []string{d.Name.Name}
	case *ast.GenDecl:
		var names []string
		for _, spec := range d.Specs {
			switch s := spec.(type) {
			case *ast.TypeSpec:
				names = append(names, s.Name.Name)
			case *ast.ValueSpec:
				for _, name := range s.Names {
					names = append(names, name.Name)
				}
			}
		}
		return names
	}
	return nil
}
