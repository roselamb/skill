package model

import "time"

type State uint8

const (
	Idle State = iota
	WalkLeft
	WalkRight
	RunLeft
	RunRight
	Jump
	Sleep
	Think
	Happy
	Error
	Drag
)

type EventKind uint8

const (
	SingleClick EventKind = iota
	DoubleClick
	DragStart
	DragMove
	DragEnd
	ContextMenu
)

type Event struct {
	Kind  EventKind
	Point Point
}

type FrameRef struct {
	Atlas string
	Row   int
	Col   int
	FlipX bool
}

type Clip struct {
	Frames    []FrameRef
	FrameTime time.Duration
	Loop      bool
}

type Point struct {
	X int
	Y int
}

type Size struct {
	W int
	H int
}

type Rect struct {
	Left   int
	Top    int
	Right  int
	Bottom int
}
