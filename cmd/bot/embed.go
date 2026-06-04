//go:build !dev

package main

import "embed"

//go:embed ui ui/_next
var embeddedUI embed.FS
