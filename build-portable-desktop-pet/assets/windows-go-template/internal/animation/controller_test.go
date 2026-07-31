package animation

import (
	"testing"
	"time"

	"portable-desktop-pet/internal/model"
)

func TestLoopingClipAdvancesAndWraps(t *testing.T) {
	controller := NewController(map[model.State]model.Clip{
		model.Idle: {
			Frames:    []model.FrameRef{{Col: 0}, {Col: 1}, {Col: 2}},
			FrameTime: 100 * time.Millisecond,
			Loop:      true,
		},
	})
	start := time.Unix(0, 0)
	controller.Play(model.Idle, start)

	tests := []struct {
		elapsed time.Duration
		wantCol int
	}{
		{elapsed: 0, wantCol: 0},
		{elapsed: 100 * time.Millisecond, wantCol: 1},
		{elapsed: 200 * time.Millisecond, wantCol: 2},
		{elapsed: 300 * time.Millisecond, wantCol: 0},
	}
	for _, test := range tests {
		frame, finished := controller.Frame(start.Add(test.elapsed))
		if frame.Col != test.wantCol {
			t.Errorf("at %v: col=%d want %d", test.elapsed, frame.Col, test.wantCol)
		}
		if finished {
			t.Errorf("at %v: looping clip unexpectedly finished", test.elapsed)
		}
	}
}

func TestNonLoopingClipStopsOnLastFrame(t *testing.T) {
	controller := NewController(map[model.State]model.Clip{
		model.Happy: {
			Frames:    []model.FrameRef{{Col: 4}, {Col: 5}, {Col: 6}},
			FrameTime: 100 * time.Millisecond,
		},
	})
	start := time.Unix(0, 0)
	controller.Play(model.Happy, start)

	frame, finished := controller.Frame(start.Add(300 * time.Millisecond))
	if frame.Col != 6 {
		t.Fatalf("col=%d want final col 6", frame.Col)
	}
	if !finished {
		t.Fatal("non-looping clip should report finished after its final frame duration")
	}
}

func TestEmptyClipIsImmediatelyFinished(t *testing.T) {
	controller := NewController(map[model.State]model.Clip{
		model.Error: {},
	})
	start := time.Unix(0, 0)
	controller.Play(model.Error, start)

	frame, finished := controller.Frame(start)
	if frame != (model.FrameRef{}) {
		t.Fatalf("frame=%v want zero value", frame)
	}
	if !finished {
		t.Fatal("empty clip should be immediately finished")
	}
}
