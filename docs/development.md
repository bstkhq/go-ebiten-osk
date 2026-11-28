# Run the demo and development checks

The module uses Go 1.25 or later and is built and tested with Ebitengine
v2.9.9. The Go API may change before the first tagged version. See
[Ebitengine's installation guide](https://ebitengine.org/en/documents/install.html)
for platform build dependencies.

## Demo controls

Follow the [checkout command](../README.md#demo) to open the
[demo](../examples/typing). Select Name, Email, or Number. Email opens the full
email layout and Number opens the number pad.

To try the [accent picker](layouts.md#accent-picker), switch to Spanish
with F1 and hold a vowel.

| Control | Effect |
| --- | --- |
| F1 | Cycle through the stock locales and keep the current field and its text. |
| F2 | Cycle through the dark theme, the light theme, and the demo's custom painter. |
| F3 | Toggle the preferred compact or full form and keep the current field and its text. |
| F4 | Show a custom PIN pad built from the exported layout blocks. |
| F7 | Show or hide the keyboard. |
| F8 | Turn the animation on or off. |

Append `-hook` to the demo command to try the
[optional Ebitengine hook](input.md#optional-ebitengine-hook).

## Development checks

Run these checks to build, vet, and verify the hook's upstream declarations.

```sh
go build ./...
go vet ./...
env -u DISPLAY -u WAYLAND_DISPLAY go test -count=1 ./hook/internal/upstream
```

The Ebitengine suites need a display. With
[go-ebiten-mcp](https://github.com/bstkhq/go-ebiten-mcp), run both variants on the GPU.

```sh
ebitenmcp run --gpu -- go test -count=1 ./...
ebitenmcp run --gpu -- go test -count=1 -tags ebitenosk_nohook ./...
```

[GitHub Actions](../.github/workflows/checks.yml) uses a private display with
software rendering. The golden pictures under `testdata` name the hardware
renderer that drew them. The tests account for renderer differences.

With golangci-lint v2.15.0 installed and `origin/main` fetched, run the same
advisory report and changed-line check as on a topic branch.

```sh
golangci-lint run --issues-exit-code=0
golangci-lint run --new-from-merge-base=origin/main
```

The changed-line check uses committed changes. On `main`, CI reports lint
findings across the module.
