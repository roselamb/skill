package assets

import (
	"fmt"
	"image"
	"image/draw"
	"time"

	"portable-desktop-pet/internal/model"
)

const (
	StandardAtlasName = "standard"
	standardColumns   = 8
	standardRows      = 11
)

// Atlas exposes fixed-size cells from a sprite sheet.
type Atlas struct {
	image      image.Image
	columns    int
	rows       int
	cellWidth  int
	cellHeight int
}

func newAtlas(source image.Image, columns, rows int) (*Atlas, error) {
	if source == nil {
		return nil, fmt.Errorf("source image is nil")
	}
	if columns <= 0 || rows <= 0 {
		return nil, fmt.Errorf("invalid grid %dx%d", columns, rows)
	}
	size := source.Bounds().Size()
	if size.X%columns != 0 || size.Y%rows != 0 {
		return nil, fmt.Errorf("image size %dx%d is not divisible by grid %dx%d", size.X, size.Y, columns, rows)
	}
	return &Atlas{
		image:      source,
		columns:    columns,
		rows:       rows,
		cellWidth:  size.X / columns,
		cellHeight: size.Y / rows,
	}, nil
}

// Bounds returns the complete atlas bounds.
func (a *Atlas) Bounds() image.Rectangle {
	if a == nil || a.image == nil {
		return image.Rectangle{}
	}
	return a.image.Bounds()
}

// Frame returns a zero-origin copy of one sprite cell.
func (a *Atlas) Frame(row, col int) (image.Image, error) {
	if !a.validCell(row, col) {
		return nil, fmt.Errorf("frame row=%d col=%d outside %dx%d atlas", row, col, a.rows, a.columns)
	}
	sourceBounds := a.image.Bounds()
	sourceRect := image.Rect(
		sourceBounds.Min.X+col*a.cellWidth,
		sourceBounds.Min.Y+row*a.cellHeight,
		sourceBounds.Min.X+(col+1)*a.cellWidth,
		sourceBounds.Min.Y+(row+1)*a.cellHeight,
	)
	frame := image.NewNRGBA(image.Rect(0, 0, a.cellWidth, a.cellHeight))
	draw.Draw(frame, frame.Bounds(), a.image, sourceRect.Min, draw.Src)
	return frame, nil
}

// OpaqueAt reports whether a frame-local pixel meets the alpha threshold.
func (a *Atlas) OpaqueAt(row, col, x, y int, threshold uint8) bool {
	if !a.validCell(row, col) || x < 0 || y < 0 || x >= a.cellWidth || y >= a.cellHeight {
		return false
	}
	bounds := a.image.Bounds()
	_, _, _, alpha := a.image.At(bounds.Min.X+col*a.cellWidth+x, bounds.Min.Y+row*a.cellHeight+y).RGBA()
	return uint8(alpha>>8) >= threshold
}

func (a *Atlas) validCell(row, col int) bool {
	return a != nil && a.image != nil && row >= 0 && row < a.rows && col >= 0 && col < a.columns
}

// ClipMap returns animation clips backed only by independent, unmirrored rows
// in the validated standard atlas.
func (b *Bundle) ClipMap() map[model.State]model.Clip {
	return map[model.State]model.Clip{
		model.Idle:      clip(0, 6, 100*time.Millisecond, true),
		model.WalkRight: clip(1, 8, 110*time.Millisecond, true),
		model.WalkLeft:  clip(2, 8, 110*time.Millisecond, true),
		model.RunRight:  clip(1, 8, 70*time.Millisecond, true),
		model.RunLeft:   clip(2, 8, 70*time.Millisecond, true),
		model.Happy:     clip(3, 4, 100*time.Millisecond, false),
		model.Jump:      clip(4, 5, 90*time.Millisecond, false),
		model.Error:     clip(5, 8, 100*time.Millisecond, false),
		model.Think:     clip(6, 6, 140*time.Millisecond, true),
		model.Sleep:     clip(0, 6, 300*time.Millisecond, true),
		model.Drag:      clip(0, 1, 100*time.Millisecond, true),
	}
}

func clip(row, frameCount int, frameTime time.Duration, loop bool) model.Clip {
	frames := make([]model.FrameRef, frameCount)
	for col := range frames {
		frames[col] = model.FrameRef{
			Atlas: StandardAtlasName,
			Row:   row,
			Col:   col,
			FlipX: false,
		}
	}
	return model.Clip{Frames: frames, FrameTime: frameTime, Loop: loop}
}
