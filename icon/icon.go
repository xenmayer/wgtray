package icon

import _ "embed"

//go:embed tray_connected.png
var connected []byte

//go:embed tray_disconnected.png
var disconnected []byte

// Connected returns PNG bytes for the "connected" (green saber) tray icon.
func Connected() []byte { return connected }

// Disconnected returns PNG bytes for the "disconnected" (grey saber) tray icon.
func Disconnected() []byte { return disconnected }

