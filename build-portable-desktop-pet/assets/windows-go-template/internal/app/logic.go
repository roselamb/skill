package app

import (
	"math"
	"time"

	"portable-desktop-pet/internal/assets"
	"portable-desktop-pet/internal/model"
	"portable-desktop-pet/internal/screen"
)

const (
	walkPixelsPerTick = 4
	runPixelsPerTick  = 8
)

type renderSnapshot struct {
	Frame    model.FrameRef
	Position model.Point
	Paused   bool
	Topmost  bool
}

func gazeFrame(center, cursor model.Point, deadRadius int) (model.FrameRef, bool) {
	dx := cursor.X - center.X
	dy := cursor.Y - center.Y
	if deadRadius < 0 {
		deadRadius = 0
	}
	if dx*dx+dy*dy <= deadRadius*deadRadius {
		return model.FrameRef{}, false
	}

	const directions = 16
	sectorWidth := 2 * math.Pi / directions
	sector := int(math.Floor(math.Atan2(float64(dy), float64(dx))/sectorWidth + 0.5))
	sector = (sector%directions + directions) % directions
	return model.FrameRef{
		Atlas: assets.StandardAtlasName,
		Row:   9 + sector/8,
		Col:   sector % 8,
		FlipX: false,
	}, true
}

func stepMotion(position, target model.Point, state model.State, size model.Size, work model.Rect) (model.Point, bool, bool) {
	speed := 0
	switch state {
	case model.WalkLeft:
		speed = -walkPixelsPerTick
	case model.WalkRight:
		speed = walkPixelsPerTick
	case model.RunLeft:
		speed = -runPixelsPerTick
	case model.RunRight:
		speed = runPixelsPerTick
	default:
		return position, true, false
	}

	next := position
	next.X += speed
	if speed > 0 && next.X >= target.X {
		next.X = target.X
	}
	if speed < 0 && next.X <= target.X {
		next.X = target.X
	}
	clamped := screen.ClampWindow(next, size, work)
	corrected := clamped != next
	reached := corrected || clamped.X == target.X
	return clamped, reached, corrected
}

func stateHoldDuration(state model.State) time.Duration {
	switch state {
	case model.Idle:
		return 1800 * time.Millisecond
	case model.Think:
		return 1600 * time.Millisecond
	case model.WalkLeft, model.WalkRight:
		return 5 * time.Second
	case model.RunLeft, model.RunRight:
		return 2500 * time.Millisecond
	case model.Happy:
		return time.Second
	default:
		return 0
	}
}

func needsRender(previous, next renderSnapshot) bool {
	return previous != next
}

func isMovementState(state model.State) bool {
	switch state {
	case model.WalkLeft, model.WalkRight, model.RunLeft, model.RunRight:
		return true
	default:
		return false
	}
}
