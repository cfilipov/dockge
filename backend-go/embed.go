package main

import "embed"

// frontendFiles embeds the built frontend for single-binary deployment.
// In dev mode (--dev flag), files are served from the filesystem instead.
//
//go:embed all:frontend-dist
var frontendFiles embed.FS
