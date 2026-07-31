package animation

import (
	"time"

	"portable-desktop-pet/internal/model"
)

type Controller struct {
	clips   map[model.State]model.Clip
	state   model.State
	started time.Time
}

func NewController(clips map[model.State]model.Clip) *Controller {
	return &Controller{clips: clips}
}

func (c *Controller) Play(state model.State, now time.Time) {
	c.state = state
	c.started = now
}

func (c *Controller) Frame(now time.Time) (model.FrameRef, bool) {
	clip := c.clips[c.state]
	if len(clip.Frames) == 0 {
		return model.FrameRef{}, true
	}
	if clip.FrameTime <= 0 {
		return clip.Frames[0], !clip.Loop
	}

	elapsed := now.Sub(c.started)
	if elapsed < 0 {
		elapsed = 0
	}
	index := int(elapsed / clip.FrameTime)
	if clip.Loop {
		return clip.Frames[index%len(clip.Frames)], false
	}
	if index >= len(clip.Frames) {
		return clip.Frames[len(clip.Frames)-1], true
	}
	return clip.Frames[index], false
}
