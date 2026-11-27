# A game reads the keyboard through the input copy; writing into Ebitengine is optional

Ebitengine has no public way to feed key presses or characters to a normal game: `ebiten.IsKeyPressed`, `ebiten.AppendInputChars` and `inpututil` read an internal state that only the window's backend fills. This holds from v2.9.9 to the main branch at the time of writing. So the supported way to read the on-screen keyboard is the input copy: functions of this library with the names and signatures of Ebitengine's, which answer for the physical keyboard and the on-screen one together. A game moves to it by changing an import.

The hook, which writes into that internal state through `//go:linkname` as go-ebiten-mcp does, is an optional package for code that cannot be changed. It is not the default because it breaks with the engine: v2.10 moves the input clock and the main branch drops one of the symbols it pulls in, so every minor version needs its own verification.

## Considered Options

- **The hook as the only way.** Nothing in the game changes, but the library would work only on the engine versions it was verified against.
- **Injection through the operating system** (`uinput`, XTest, `SendInput`). It needs permissions on the device, exists per platform, and the text that arrives depends on the system's configured layout, not on the one the screen shows.
- **Ebitengine 2.10's `exp/vmhost` guest.** It is upstream's supported route, but the game must run as the guest of another process, which changes how every game starts.
