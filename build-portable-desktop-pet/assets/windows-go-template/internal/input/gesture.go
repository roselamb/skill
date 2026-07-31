package input

import (
	"time"

	"portable-desktop-pet/internal/model"
)

type GestureDetector struct {
	doubleClick time.Duration
	thresholdSq int

	down      bool
	downPoint model.Point
	dragging  bool

	pending         bool
	pendingPoint    model.Point
	pendingDeadline time.Time
	doubleCandidate bool
}

func NewGestureDetector(doubleClick time.Duration, dragThreshold int) *GestureDetector {
	return &GestureDetector{
		doubleClick: doubleClick,
		thresholdSq: dragThreshold * dragThreshold,
	}
}

func (d *GestureDetector) Down(point model.Point, now time.Time) []model.Event {
	if d.down {
		return nil
	}

	var events []model.Event
	if d.pending && !now.Before(d.pendingDeadline) {
		events = []model.Event{{Kind: model.SingleClick, Point: d.pendingPoint}}
		d.pending = false
		d.doubleCandidate = false
	}

	d.down = true
	d.downPoint = point
	d.dragging = false
	d.doubleCandidate = d.pending && !now.After(d.pendingDeadline)
	return events
}

func (d *GestureDetector) Move(point model.Point, _ time.Time) []model.Event {
	if !d.down {
		return nil
	}
	if d.dragging {
		return []model.Event{{Kind: model.DragMove, Point: point}}
	}

	dx := point.X - d.downPoint.X
	dy := point.Y - d.downPoint.Y
	if dx*dx+dy*dy <= d.thresholdSq {
		return nil
	}

	d.dragging = true
	d.pending = false
	d.doubleCandidate = false
	return []model.Event{{Kind: model.DragStart, Point: point}}
}

func (d *GestureDetector) Up(point model.Point, now time.Time) []model.Event {
	if !d.down {
		return nil
	}
	d.down = false

	if d.dragging {
		d.dragging = false
		return []model.Event{{Kind: model.DragEnd, Point: point}}
	}

	if d.doubleCandidate {
		d.pending = false
		d.doubleCandidate = false
		return []model.Event{{Kind: model.DoubleClick, Point: point}}
	}

	d.pending = true
	d.pendingPoint = point
	d.pendingDeadline = now.Add(d.doubleClick)
	return nil
}

func (d *GestureDetector) Tick(now time.Time) []model.Event {
	if !d.pending || d.doubleCandidate || now.Before(d.pendingDeadline) {
		return nil
	}
	d.pending = false
	return []model.Event{{Kind: model.SingleClick, Point: d.pendingPoint}}
}

func (d *GestureDetector) Cancel(point model.Point) []model.Event {
	if !d.down {
		return nil
	}

	d.down = false
	d.doubleCandidate = false
	if !d.dragging {
		return nil
	}

	d.dragging = false
	return []model.Event{{Kind: model.DragEnd, Point: point}}
}
