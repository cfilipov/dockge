package main

import "embed"

// staticFiles embeds the built frontend for single-binary deployment.
// In dev mode (--dev flag), files are served from the filesystem instead.
//
//go:embed all:dist
var staticFiles embed.FS
