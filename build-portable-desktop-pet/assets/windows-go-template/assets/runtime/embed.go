package runtimeassets

import _ "embed"

// SpritesheetPNG is the validated 8 by 11 animation atlas.
//
//go:embed spritesheet.png
var SpritesheetPNG []byte

// AppIconPNG is the transparent application icon.
//
//go:embed app-icon.png
var AppIconPNG []byte
