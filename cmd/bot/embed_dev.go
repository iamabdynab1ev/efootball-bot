//go:build dev

package main

import "embed"

// embeddedUI is empty in dev mode — Next.js serves the frontend separately.
var embeddedUI embed.FS
