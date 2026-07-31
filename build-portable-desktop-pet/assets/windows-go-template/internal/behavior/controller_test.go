package behavior

import (
	"testing"
	"time"

	"portable-desktop-pet/internal/model"
)

func TestDragPreemptsHappyAndAuto(t *testing.T) {
	c := New(7)
	c.Request(model.Event{Kind: model.SingleClick}, time.Unix(0, 0))

	got := c.Request(model.Event{Kind: model.DragStart}, time.Unix(1, 0))

	if got != model.Drag {
		t.Fatalf("got=%v want Drag", got)
	}
	if decision := c.NextAuto(time.Unix(2, 0), model.Rect{Right: 100, Bottom: 100}); decision.State != model.Drag {
		t.Fatalf("automatic decision during drag=%v want Drag", decision.State)
	}
}

func TestDoubleClickTogglesSleep(t *testing.T) {
	c := New(7)
	now := time.Unix(0, 0)

	if got := c.Request(model.Event{Kind: model.DoubleClick}, now); got != model.Sleep {
		t.Fatalf("first DoubleClick state=%v want Sleep", got)
	}
	if got := c.Request(model.Event{Kind: model.DoubleClick}, now.Add(time.Second)); got != model.Idle {
		t.Fatalf("second DoubleClick state=%v want Idle", got)
	}
}

func TestPausedControllerDoesNotScheduleAutomaticActivity(t *testing.T) {
	c := New(7)
	c.SetPaused(true)
	bounds := model.Rect{Left: -200, Top: -100, Right: 300, Bottom: 400}

	for i := 0; i < 100; i++ {
		decision := c.NextAuto(time.Unix(int64(i), 0), bounds)
		switch decision.State {
		case model.WalkLeft, model.WalkRight, model.RunLeft, model.RunRight, model.Think:
			t.Fatalf("paused automatic decision %d state=%v", i, decision.State)
		}
	}
}

func TestRunIsScheduledLessOftenThanWalk(t *testing.T) {
	c := New(7)
	bounds := model.Rect{Left: -200, Top: -100, Right: 300, Bottom: 400}
	var walks, runs int

	for i := 0; i < 1000; i++ {
		switch c.NextAuto(time.Unix(int64(i), 0), bounds).State {
		case model.WalkLeft, model.WalkRight:
			walks++
		case model.RunLeft, model.RunRight:
			runs++
		}
	}

	if runs == 0 {
		t.Fatal("deterministic sequence never scheduled Run")
	}
	if walks == 0 {
		t.Fatal("deterministic sequence never scheduled Walk")
	}
	if runs >= walks {
		t.Fatalf("runs=%d walks=%d want Run less frequent than Walk", runs, walks)
	}
}

func TestAutomaticTargetStaysInsideBounds(t *testing.T) {
	c := New(19)
	c.SetPosition(model.Point{X: 90, Y: 300}, model.Size{W: 80, H: 100})
	bounds := model.Rect{Left: -320, Top: 40, Right: 180, Bottom: 440}

	for i := 0; i < 500; i++ {
		decision := c.NextAuto(time.Unix(int64(i), 0), bounds)
		if !decision.HasTarget {
			continue
		}
		if decision.Target.X < bounds.Left || decision.Target.X > bounds.Right-80 ||
			decision.Target.Y < bounds.Top || decision.Target.Y > bounds.Bottom-100 {
			t.Fatalf("decision %d target=%v outside bounds=%v", i, decision.Target, bounds)
		}
	}
}

func TestAutomaticMovementStaysNearAnchorAndFacesTarget(t *testing.T) {
	c := New(19)
	current := model.Point{X: 90, Y: 300}
	c.SetPosition(current, model.Size{W: 80, H: 100})
	bounds := model.Rect{Left: -50, Top: 0, Right: 250, Bottom: 600}
	movementCount := 0

	for i := 0; i < 500; i++ {
		decision := c.NextAuto(time.Unix(int64(i), 0), bounds)
		if !decision.HasTarget {
			continue
		}
		movementCount++
		if decision.Target.Y < 236 || decision.Target.Y > 364 {
			t.Fatalf("decision %d target Y=%d want within 64 px of anchor 300", i, decision.Target.Y)
		}
		switch {
		case decision.Target.X < current.X:
			if decision.State != model.WalkLeft && decision.State != model.RunLeft {
				t.Fatalf("decision %d target=%v state=%v want left-facing movement", i, decision.Target, decision.State)
			}
		case decision.Target.X > current.X:
			if decision.State != model.WalkRight && decision.State != model.RunRight {
				t.Fatalf("decision %d target=%v state=%v want right-facing movement", i, decision.Target, decision.State)
			}
		default:
			t.Fatalf("decision %d movement target X equals current X=%d", i, current.X)
		}
	}
	if movementCount == 0 {
		t.Fatal("deterministic sequence produced no movement decisions")
	}
}

func TestAutomaticVerticalBandClampsAtWorkAreaEdge(t *testing.T) {
	c := New(23)
	c.SetPosition(model.Point{X: 10, Y: 580}, model.Size{W: 80, H: 100})
	bounds := model.Rect{Left: 0, Top: 0, Right: 300, Bottom: 600}

	for i := 0; i < 100; i++ {
		decision := c.NextAuto(time.Unix(int64(i), 0), bounds)
		if decision.HasTarget && decision.Target.Y != 500 {
			t.Fatalf("decision %d target Y=%d want clamped bottom position 500", i, decision.Target.Y)
		}
	}
}

func TestPositionUpdatesPreserveDragAnchorAcrossAutomaticMoves(t *testing.T) {
	c := New(41)
	size := model.Size{W: 80, H: 100}
	current := model.Point{X: 90, Y: 300}
	c.SetPosition(current, size)
	bounds := model.Rect{Left: -50, Top: 250, Right: 300, Bottom: 420}
	movementCount := 0

	for i := 0; i < 500 && movementCount < 100; i++ {
		decision := c.NextAuto(time.Unix(int64(i), 0), bounds)
		if !decision.HasTarget {
			continue
		}
		if decision.Target.X < bounds.Left || decision.Target.X > bounds.Right-size.W ||
			decision.Target.Y < 250 || decision.Target.Y > 320 {
			t.Fatalf("decision %d target=%v outside anchored clamped area", i, decision.Target)
		}
		if decision.Target.X < current.X {
			if decision.State != model.WalkLeft && decision.State != model.RunLeft {
				t.Fatalf("decision %d current=%v target=%v state=%v want left", i, current, decision.Target, decision.State)
			}
		} else if decision.Target.X > current.X {
			if decision.State != model.WalkRight && decision.State != model.RunRight {
				t.Fatalf("decision %d current=%v target=%v state=%v want right", i, current, decision.Target, decision.State)
			}
		} else {
			t.Fatalf("decision %d target X equals latest current X=%d", i, current.X)
		}

		current = decision.Target
		c.UpdatePosition(current, size)
		movementCount++
	}
	if movementCount != 100 {
		t.Fatalf("movement decisions=%d want 100", movementCount)
	}
}

func TestFirstPositionUpdateInitializesAnchor(t *testing.T) {
	c := New(43)
	c.UpdatePosition(model.Point{X: 10, Y: 580}, model.Size{W: 80, H: 100})
	bounds := model.Rect{Left: 0, Top: 0, Right: 300, Bottom: 600}

	for i := 0; i < 100; i++ {
		decision := c.NextAuto(time.Unix(int64(i), 0), bounds)
		if decision.HasTarget && decision.Target.Y != 500 {
			t.Fatalf("decision %d target Y=%d want anchor initialized and clamped to 500", i, decision.Target.Y)
		}
	}
}

func TestSameSeedProducesSameAutomaticSequence(t *testing.T) {
	first := New(31)
	second := New(31)
	position := model.Point{X: 40, Y: 120}
	size := model.Size{W: 80, H: 100}
	first.SetPosition(position, size)
	second.SetPosition(position, size)
	bounds := model.Rect{Left: -100, Top: 0, Right: 400, Bottom: 500}

	for i := 0; i < 200; i++ {
		now := time.Unix(int64(i), 0)
		got := first.NextAuto(now, bounds)
		want := second.NextAuto(now, bounds)
		if got != want {
			t.Fatalf("decision %d got=%+v want=%+v", i, got, want)
		}
	}
}

func TestHappyReactionPreemptsAutomaticActivity(t *testing.T) {
	c := New(7)
	start := time.Unix(0, 0)
	if got := c.Request(model.Event{Kind: model.SingleClick}, start); got != model.Happy {
		t.Fatalf("SingleClick state=%v want Happy", got)
	}

	decision := c.NextAuto(start.Add(500*time.Millisecond), model.Rect{Right: 100, Bottom: 100})
	if decision.State != model.Happy || decision.HasTarget {
		t.Fatalf("automatic decision during Happy=%+v want Happy without target", decision)
	}
}
