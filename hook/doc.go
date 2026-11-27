// Package hook lets existing Ebitengine input readers see on-screen keyboard
// input. It uses Ebitengine's internal input snapshot and supports release
// versions in the 2.9 series. Build with ebitenosk_nohook to omit the engine
// integration.
//
// A physically held key keeps its own press timestamp. If it takes over an
// on-screen hold, Ebitengine can report a new press and restart the duration.
package hook
