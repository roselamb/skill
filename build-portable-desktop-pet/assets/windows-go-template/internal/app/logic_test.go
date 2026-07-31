package app

import (
	"math"
	"testing"
	"time"

	"portable-desktop-pet/internal/model"
)

func gazeTestPoint(center model.Point, sector, distance int) model.Point {
	angle := float64(sector) * 2 * math.Pi / 16
	return model.Point{
		X: center.X + int(math.Round(math.Cos(angle)*float64(distance))),
		Y: center.Y + int(math.Round(math.Sin(angle)*float64(distance))),
	}
}

func TestGazeFrameQuantizesAllCardinalDirections(t *testing.T) {
	center := model.Point{X: 100, Y: 100}
	tests := []struct {
		name   string
		cursor model.Point
		want   model.FrameRef
	}{
		{name: "right", cursor: model.Point{X: 200, Y: 100}, want: model.FrameRef{Atlas: "standard", Row: 9, Col: 0}},
		{name: "down", cursor: model.Point{X: 100, Y: 200}, want: model.FrameRef{Atlas: "standard", Row: 9, Col: 4}},
		{name: "left", cursor: model.Point{X: 0, Y: 100}, want: model.FrameRef{Atlas: "standard", Row: 10, Col: 0}},
		{name: "up", cursor: model.Point{X: 100, Y: 0}, want: model.FrameRef{Atlas: "standard", Row: 10, Col: 4}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, ok := gazeFrame(center, test.cursor, 20)
			if !ok {
				t.Fatal("gazeFrame reported center dead zone")
			}
			if got != test.want {
				t.Fatalf("frame=%+v want %+v", got, test.want)
			}
			if got.FlipX {
				t.Fatal("gaze frame must use an approved unmirrored atlas cell")
			}
		})
	}
}

func TestGazeFrameUsesDeadZoneAndAllSixteenCells(t *testing.T) {
	center := model.Point{X: 500, Y: 500}
	if _, ok := gazeFrame(center, model.Point{X: 510, Y: 510}, 20); ok {
		t.Fatal("cursor inside center dead zone should not override Idle")
	}

	seen := make(map[[2]int]bool)
	for i := 0; i < 16; i++ {
		cursor := gazeTestPoint(center, i, 200)
		frame, ok := gazeFrame(center, cursor, 20)
		if !ok {
			t.Fatalf("direction %d unexpectedly in dead zone", i)
		}
		seen[[2]int{frame.Row, frame.Col}] = true
	}
	if len(seen) != 16 {
		t.Fatalf("unique gaze cells=%d want 16", len(seen))
	}
}

func TestStepMotionUsesStateSpeedAndClampsAtWorkArea(t *testing.T) {
	work := model.Rect{Left: -300, Top: 0, Right: 300, Bottom: 240}
	size := model.Size{W: 100, H: 100}

	walk, reached, corrected := stepMotion(
		model.Point{X: 0, Y: 80},
		model.Point{X: 200, Y: 120},
		model.WalkRight,
		size,
		work,
	)
	if walk != (model.Point{X: 4, Y: 80}) || reached || corrected {
		t.Fatalf("walk got=%+v reached=%v corrected=%v", walk, reached, corrected)
	}

	run, _, _ := stepMotion(
		model.Point{X: 0, Y: 80},
		model.Point{X: 200, Y: 120},
		model.RunRight,
		size,
		work,
	)
	if run != (model.Point{X: 8, Y: 80}) {
		t.Fatalf("run got=%+v want x=8 y=80", run)
	}

	clamped, reached, corrected := stepMotion(
		model.Point{X: 198, Y: 80},
		model.Point{X: 500, Y: 80},
		model.RunRight,
		size,
		work,
	)
	if clamped != (model.Point{X: 200, Y: 80}) || !reached || !corrected {
		t.Fatalf("boundary got=%+v reached=%v corrected=%v", clamped, reached, corrected)
	}
}

func TestStepMotionDoesNotMoveForNonMovementStates(t *testing.T) {
	position := model.Point{X: 12, Y: 34}
	got, reached, corrected := stepMotion(
		position,
		model.Point{X: 200, Y: 34},
		model.Happy,
		model.Size{W: 100, H: 100},
		model.Rect{Right: 500, Bottom: 500},
	)
	if got != position || !reached || corrected {
		t.Fatalf("got=%+v reached=%v corrected=%v", got, reached, corrected)
	}
}

func TestStateHoldDurationIsDeterministic(t *testing.T) {
	tests := map[model.State]time.Duration{
		model.Idle:      1800 * time.Millisecond,
		model.Think:     1600 * time.Millisecond,
		model.WalkLeft:  5 * time.Second,
		model.WalkRight: 5 * time.Second,
		model.RunLeft:   2500 * time.Millisecond,
		model.RunRight:  2500 * time.Millisecond,
		model.Happy:     time.Second,
		model.Sleep:     0,
		model.Drag:      0,
	}
	for state, want := range tests {
		if got := stateHoldDuration(state); got != want {
			t.Fatalf("state=%v duration=%v want %v", state, got, want)
		}
	}
}

func TestRenderDecisionTracksOnlyVisibleInputs(t *testing.T) {
	base := renderSnapshot{
		Frame:    model.FrameRef{Atlas: "standard", Row: 0, Col: 1},
		Position: model.Point{X: 10, Y: 20},
		Paused:   false,
		Topmost:  true,
	}
	if needsRender(base, base) {
		t.Fatal("unchanged snapshot should not render")
	}

	changes := []renderSnapshot{
		{Frame: model.FrameRef{Atlas: "standard", Row: 0, Col: 2}, Position: base.Position, Topmost: true},
		{Frame: base.Frame, Position: model.Point{X: 11, Y: 20}, Topmost: true},
		{Frame: base.Frame, Position: base.Position, Paused: true, Topmost: true},
		{Frame: base.Frame, Position: base.Position, Topmost: false},
	}
	for i, change := range changes {
		if !needsRender(base, change) {
			t.Fatalf("visible change %d was not marked dirty", i)
		}
	}
}
