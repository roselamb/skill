package behavior

import (
	"math/rand"
	"time"

	"portable-desktop-pet/internal/model"
)

const happyDuration = time.Second
const verticalRoam = 64

type Decision struct {
	State     model.State
	Target    model.Point
	HasTarget bool
}

type Controller struct {
	random *rand.Rand

	state         model.State
	paused        bool
	sleeping      bool
	dragging      bool
	reactionUntil time.Time
	current       model.Point
	size          model.Size
	anchorY       int
	hasAnchor     bool
}

func New(seed int64) *Controller {
	return &Controller{
		random: rand.New(rand.NewSource(seed)),
		state:  model.Idle,
	}
}

func (c *Controller) Request(event model.Event, now time.Time) model.State {
	switch event.Kind {
	case model.DragStart, model.DragMove:
		c.dragging = true
		c.state = model.Drag
		return c.state
	case model.DragEnd:
		c.dragging = false
		if c.sleeping {
			c.state = model.Sleep
		} else {
			c.state = model.Idle
		}
		return c.state
	}

	if c.dragging {
		return model.Drag
	}

	switch event.Kind {
	case model.SingleClick:
		c.state = model.Happy
		c.reactionUntil = now.Add(happyDuration)
	case model.DoubleClick:
		c.sleeping = !c.sleeping
		c.reactionUntil = time.Time{}
		if c.sleeping {
			c.state = model.Sleep
		} else {
			c.state = model.Idle
		}
	}
	return c.state
}

func (c *Controller) SetPaused(paused bool) {
	c.paused = paused
	if paused && isAutomatic(c.state) {
		c.state = model.Idle
	}
}

func (c *Controller) SetPosition(position model.Point, size model.Size) {
	c.current = position
	c.size = normalizedSize(size)
	c.anchorY = position.Y
	c.hasAnchor = true
}

func (c *Controller) UpdatePosition(position model.Point, size model.Size) {
	c.current = position
	c.size = normalizedSize(size)
	if !c.hasAnchor {
		c.anchorY = position.Y
		c.hasAnchor = true
	}
}

func (c *Controller) NextAuto(now time.Time, bounds model.Rect) Decision {
	if c.dragging {
		return Decision{State: model.Drag}
	}
	if now.Before(c.reactionUntil) {
		return Decision{State: c.state}
	}
	if c.sleeping {
		c.state = model.Sleep
		return Decision{State: c.state}
	}
	if c.paused {
		c.state = model.Idle
		return Decision{State: c.state}
	}

	roll := c.random.Intn(100)
	switch {
	case roll < 55:
		return c.movementDecision(bounds, false)
	case roll < 65:
		return c.movementDecision(bounds, true)
	case roll < 80:
		c.state = model.Think
	default:
		c.state = model.Idle
	}
	return Decision{State: c.state}
}

func (c *Controller) movementDecision(bounds model.Rect, running bool) Decision {
	minX := bounds.Left
	maxX := bounds.Right - c.size.W
	minY := bounds.Top
	maxY := bounds.Bottom - c.size.H
	if maxX < minX || maxY < minY {
		c.state = model.Idle
		return Decision{State: c.state}
	}

	bandMinY := clamp(c.anchorY-verticalRoam, minY, maxY)
	bandMaxY := clamp(c.anchorY+verticalRoam, minY, maxY)
	target := model.Point{
		X: c.randomBetween(minX, maxX),
		Y: c.randomBetween(bandMinY, bandMaxY),
	}
	if target.X == c.current.X {
		switch {
		case target.X < maxX:
			target.X++
		case target.X > minX:
			target.X--
		default:
			c.state = model.Idle
			return Decision{State: c.state}
		}
	}

	if target.X < c.current.X {
		if running {
			c.state = model.RunLeft
		} else {
			c.state = model.WalkLeft
		}
	} else if running {
		c.state = model.RunRight
	} else {
		c.state = model.WalkRight
	}
	return Decision{State: c.state, Target: target, HasTarget: true}
}

func (c *Controller) randomBetween(minimum, maximum int) int {
	if maximum <= minimum {
		return minimum
	}
	return minimum + c.random.Intn(maximum-minimum+1)
}

func clamp(value, minimum, maximum int) int {
	if value < minimum {
		return minimum
	}
	if value > maximum {
		return maximum
	}
	return value
}

func normalizedSize(size model.Size) model.Size {
	if size.W < 0 {
		size.W = 0
	}
	if size.H < 0 {
		size.H = 0
	}
	return size
}

func isAutomatic(state model.State) bool {
	switch state {
	case model.WalkLeft, model.WalkRight, model.RunLeft, model.RunRight, model.Think:
		return true
	default:
		return false
	}
}
