package win32

import (
	"image"
	"image/color"
	"testing"

	"portable-desktop-pet/internal/model"
)

func TestPrepareFrameConvertsToPremultipliedBGRA(t *testing.T) {
	src := image.NewNRGBA(image.Rect(0, 0, 2, 1))
	src.SetNRGBA(0, 0, color.NRGBA{R: 100, G: 50, B: 200, A: 128})
	src.SetNRGBA(1, 0, color.NRGBA{R: 9, G: 8, B: 7, A: 0})

	got := prepareFrame(src)

	want := []byte{100, 25, 50, 128, 0, 0, 0, 0}
	if string(got.Pixels) != string(want) {
		t.Fatalf("pixels=%v want %v", got.Pixels, want)
	}
}

func TestPrepareFrameUsesTopDownRowsAndDWORDStride(t *testing.T) {
	src := image.NewNRGBA(image.Rect(4, 7, 5, 9))
	src.SetNRGBA(4, 7, color.NRGBA{R: 1, A: 255})
	src.SetNRGBA(4, 8, color.NRGBA{B: 2, A: 255})

	got := prepareFrame(src)

	if got.Width != 1 || got.Height != 2 || got.Stride != 4 {
		t.Fatalf("geometry=%dx%d stride=%d want 1x2 stride=4", got.Width, got.Height, got.Stride)
	}
	if got.Pixels[2] != 1 || got.Pixels[4] != 2 {
		t.Fatalf("rows are not stored top-down: %v", got.Pixels)
	}
}

func TestAlphaMaskHitUsesBoundsAndThreshold(t *testing.T) {
	mask := alphaMask{
		Width:  2,
		Height: 2,
		Stride: 2,
		Pixels: []byte{0, 15, 16, 255},
	}

	for _, point := range []image.Point{{-1, 0}, {0, -1}, {2, 0}, {0, 2}, {0, 0}, {1, 0}} {
		if mask.Hit(point.X, point.Y) {
			t.Fatalf("Hit(%v)=true want false", point)
		}
	}
	if !mask.Hit(0, 1) || !mask.Hit(1, 1) {
		t.Fatalf("alpha >= 16 should be hittable")
	}
}

func TestAlphaMaskHitWithinAddsForgivingClickSlop(t *testing.T) {
	mask := alphaMask{Width: 3, Height: 1, Stride: 3, Pixels: []byte{0, 0, 255}}
	if !mask.HitWithin(1, 0, 1) {
		t.Fatal("one-pixel click slop should reach the visible sprite edge")
	}
	if mask.HitWithin(0, 0, 0) {
		t.Fatal("zero slop should preserve exact alpha hit behavior")
	}
}

func TestPreparedFrameCarriesMatchingAlphaMask(t *testing.T) {
	src := image.NewAlpha(image.Rect(10, 10, 12, 11))
	src.SetAlpha(10, 10, color.Alpha{A: 15})
	src.SetAlpha(11, 10, color.Alpha{A: 16})

	got := prepareFrame(src)

	if got.Mask.Hit(0, 0) || !got.Mask.Hit(1, 0) {
		t.Fatalf("mask did not preserve source alpha threshold")
	}
}

func TestScaleFrameFitsTargetAndPreservesAspectRatio(t *testing.T) {
	src := image.NewNRGBA(image.Rect(0, 0, 2, 4))

	got := scaleFrame(src, model.Size{W: 190, H: 190})

	if got.Bounds().Dx() != 95 || got.Bounds().Dy() != 190 {
		t.Fatalf("scaled=%dx%d want 95x190", got.Bounds().Dx(), got.Bounds().Dy())
	}
}

func TestScaleFramePreservesHardAlphaEdges(t *testing.T) {
	src := image.NewNRGBA(image.Rect(0, 0, 2, 1))
	src.SetNRGBA(0, 0, color.NRGBA{R: 255, A: 255})
	src.SetNRGBA(1, 0, color.NRGBA{B: 255, A: 0})

	got := scaleFrame(src, model.Size{W: 4, H: 2})

	for y := 0; y < 2; y++ {
		if got.NRGBAAt(0, y).A != 255 || got.NRGBAAt(1, y).A != 255 {
			t.Fatalf("opaque half lost alpha at row %d", y)
		}
		if got.NRGBAAt(2, y).A != 0 || got.NRGBAAt(3, y).A != 0 {
			t.Fatalf("transparent half gained alpha at row %d", y)
		}
	}
}
