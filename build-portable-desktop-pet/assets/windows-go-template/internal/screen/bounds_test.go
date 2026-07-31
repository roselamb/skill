package screen

import (
	"testing"

	"portable-desktop-pet/internal/model"
)

func TestClampWindowMovesTopLeftInsideWorkArea(t *testing.T) {
	got := ClampWindow(
		model.Point{X: -20, Y: -30},
		model.Size{W: 200, H: 150},
		model.Rect{Left: 0, Top: 0, Right: 1920, Bottom: 1080},
	)
	want := model.Point{X: 0, Y: 0}
	if got != want {
		t.Fatalf("got=%v want=%v", got, want)
	}
}

func TestClampWindowMovesBottomRightInsideWorkArea(t *testing.T) {
	got := ClampWindow(
		model.Point{X: 1850, Y: 1000},
		model.Size{W: 200, H: 150},
		model.Rect{Left: 0, Top: 0, Right: 1920, Bottom: 1080},
	)
	want := model.Point{X: 1720, Y: 930}
	if got != want {
		t.Fatalf("got=%v want=%v", got, want)
	}
}

func TestClampWindowAnchorsOversizedWindowAtWorkAreaOrigin(t *testing.T) {
	got := ClampWindow(
		model.Point{X: 500, Y: 500},
		model.Size{W: 2500, H: 1200},
		model.Rect{Left: 100, Top: 50, Right: 2020, Bottom: 1130},
	)
	want := model.Point{X: 100, Y: 50}
	if got != want {
		t.Fatalf("got=%v want=%v", got, want)
	}
}

func TestClampWindowOnNegativeMonitor(t *testing.T) {
	got := ClampWindow(
		model.Point{X: -2100, Y: 900},
		model.Size{W: 200, H: 200},
		model.Rect{Left: -1920, Top: 0, Right: 0, Bottom: 1080},
	)
	want := model.Point{X: -1920, Y: 880}
	if got != want {
		t.Fatalf("got=%v want=%v", got, want)
	}
}
