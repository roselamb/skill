package assets

import (
	"bytes"
	"fmt"
	"image"
	_ "image/png"

	runtimeassets "portable-desktop-pet/assets/runtime"
)

// Bundle contains all image assets required by the portable application.
type Bundle struct {
	Standard *Atlas
	Icon     image.Image
}

// Load decodes the assets embedded in the executable.
func Load() (*Bundle, error) {
	spritesheet, _, err := image.Decode(bytes.NewReader(runtimeassets.SpritesheetPNG))
	if err != nil {
		return nil, fmt.Errorf("decode standard atlas: %w", err)
	}
	icon, _, err := image.Decode(bytes.NewReader(runtimeassets.AppIconPNG))
	if err != nil {
		return nil, fmt.Errorf("decode application icon: %w", err)
	}

	atlas, err := newAtlas(spritesheet, 8, 11)
	if err != nil {
		return nil, fmt.Errorf("load standard atlas: %w", err)
	}
	return &Bundle{Standard: atlas, Icon: icon}, nil
}
