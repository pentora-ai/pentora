package ui

import (
	"embed"
)

// Files starting with . and _ are excluded by default
//
//go:embed dist/*
var Assets embed.FS
