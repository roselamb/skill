package screen

import "portable-desktop-pet/internal/model"

func ClampWindow(pos model.Point, size model.Size, work model.Rect) model.Point {
	return model.Point{
		X: clampAxis(pos.X, size.W, work.Left, work.Right),
		Y: clampAxis(pos.Y, size.H, work.Top, work.Bottom),
	}
}

func clampAxis(position, windowSize, lower, upper int) int {
	maximum := upper - windowSize
	if maximum < lower {
		return lower
	}
	if position < lower {
		return lower
	}
	if position > maximum {
		return maximum
	}
	return position
}
