package assets

import (
	"image"
	"testing"

	"portable-desktop-pet/internal/model"
)

func TestLoadDecodesEmbeddedRuntimeAssets(t *testing.T) {
	bundle, err := Load()
	if err != nil {
		t.Fatal(err)
	}

	if bundle.Standard == nil {
		t.Fatal("standard atlas is nil")
	}
	if got, want := bundle.Standard.Bounds().Size(), (image.Point{X: 1536, Y: 2288}); got != want {
		t.Fatalf("standard atlas size=%v want=%v", got, want)
	}
	if bundle.Icon == nil {
		t.Fatal("icon is nil")
	}
	if got := bundle.Icon.Bounds().Size(); got.X <= 0 || got.Y <= 0 {
		t.Fatalf("icon has invalid size=%v", got)
	}
}

func TestFrameReturnsOneCellAndRejectsOutOfRangeCoordinates(t *testing.T) {
	bundle := mustLoad(t)

	frame, err := bundle.Standard.Frame(10, 7)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := frame.Bounds().Size(), (image.Point{X: 192, Y: 208}); got != want {
		t.Fatalf("frame size=%v want=%v", got, want)
	}

	for _, tc := range []struct {
		row int
		col int
	}{
		{row: -1, col: 0},
		{row: 0, col: -1},
		{row: 11, col: 0},
		{row: 0, col: 8},
	} {
		if _, err := bundle.Standard.Frame(tc.row, tc.col); err == nil {
			t.Fatalf("Frame(%d, %d) succeeded; want error", tc.row, tc.col)
		}
	}
}

func TestOpaqueAtUsesFrameLocalAlphaAndIsSafeOutsideTheCell(t *testing.T) {
	atlas := mustLoad(t).Standard

	if atlas.OpaqueAt(0, 0, 0, 0, 1) {
		t.Fatal("transparent frame corner reported opaque")
	}
	if !atlas.OpaqueAt(0, 0, 96, 104, 128) {
		t.Fatal("opaque character center reported transparent")
	}

	for _, tc := range []struct {
		row int
		col int
		x   int
		y   int
	}{
		{row: -1, col: 0, x: 0, y: 0},
		{row: 0, col: 8, x: 0, y: 0},
		{row: 0, col: 0, x: -1, y: 0},
		{row: 0, col: 0, x: 192, y: 0},
		{row: 0, col: 0, x: 0, y: 208},
	} {
		if atlas.OpaqueAt(tc.row, tc.col, tc.x, tc.y, 1) {
			t.Fatalf("OpaqueAt(%d, %d, %d, %d) outside cell reported opaque", tc.row, tc.col, tc.x, tc.y)
		}
	}
}

func TestClipMapCoversEveryStateWithIndependentUnflippedFrames(t *testing.T) {
	clips := mustLoad(t).ClipMap()
	wantRows := map[model.State]int{
		model.Idle:      0,
		model.WalkRight: 1,
		model.WalkLeft:  2,
		model.RunRight:  1,
		model.RunLeft:   2,
		model.Happy:     3,
		model.Jump:      4,
		model.Error:     5,
		model.Think:     6,
		model.Sleep:     0,
		model.Drag:      0,
	}

	if got, want := len(clips), len(wantRows); got != want {
		t.Fatalf("clip count=%d want=%d", got, want)
	}
	for state, wantRow := range wantRows {
		clip, ok := clips[state]
		if !ok {
			t.Fatalf("missing clip for state %v", state)
		}
		if len(clip.Frames) == 0 {
			t.Fatalf("state %v has an empty clip", state)
		}
		for _, frame := range clip.Frames {
			if frame.Atlas != StandardAtlasName {
				t.Fatalf("state %v atlas=%q want=%q", state, frame.Atlas, StandardAtlasName)
			}
			if frame.Row != wantRow {
				t.Fatalf("state %v row=%d want=%d", state, frame.Row, wantRow)
			}
			if frame.Col < 0 || frame.Col >= 8 {
				t.Fatalf("state %v has out-of-range col=%d", state, frame.Col)
			}
			if frame.FlipX {
				t.Fatalf("state %v uses forbidden horizontal mirroring", state)
			}
		}
	}
}

func TestSimplifiedClipTimingAndDragFrame(t *testing.T) {
	clips := mustLoad(t).ClipMap()

	if clips[model.RunRight].FrameTime >= clips[model.WalkRight].FrameTime {
		t.Fatalf("run-right frame time=%v must be faster than walk-right=%v",
			clips[model.RunRight].FrameTime, clips[model.WalkRight].FrameTime)
	}
	if clips[model.RunLeft].FrameTime >= clips[model.WalkLeft].FrameTime {
		t.Fatalf("run-left frame time=%v must be faster than walk-left=%v",
			clips[model.RunLeft].FrameTime, clips[model.WalkLeft].FrameTime)
	}
	if clips[model.Sleep].FrameTime <= clips[model.Idle].FrameTime {
		t.Fatalf("sleep frame time=%v must be slower than idle=%v",
			clips[model.Sleep].FrameTime, clips[model.Idle].FrameTime)
	}
	if got := len(clips[model.Drag].Frames); got != 1 {
		t.Fatalf("drag frame count=%d want=1", got)
	}
}

func mustLoad(t *testing.T) *Bundle {
	t.Helper()
	bundle, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	return bundle
}
