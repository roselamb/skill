package input

import (
	"testing"
	"time"

	"portable-desktop-pet/internal/model"
)

func TestSingleClickWaitsForDoubleClickWindowAndFiresOnce(t *testing.T) {
	detector := NewGestureDetector(250*time.Millisecond, 4)
	start := time.Unix(0, 0)
	point := model.Point{X: 20, Y: 30}

	detector.Down(point, start)
	if events := detector.Up(point, start.Add(20*time.Millisecond)); len(events) != 0 {
		t.Fatalf("first Up events=%v want none", events)
	}
	if events := detector.Tick(start.Add(249 * time.Millisecond)); len(events) != 0 {
		t.Fatalf("Tick before deadline events=%v want none", events)
	}

	events := detector.Tick(start.Add(270 * time.Millisecond))
	if len(events) != 1 || events[0].Kind != model.SingleClick || events[0].Point != point {
		t.Fatalf("Tick after deadline events=%v want one SingleClick at %v", events, point)
	}
	if events := detector.Tick(start.Add(time.Second)); len(events) != 0 {
		t.Fatalf("later Tick events=%v want none", events)
	}
}

func TestSecondClickCancelsPendingSingleClick(t *testing.T) {
	detector := NewGestureDetector(250*time.Millisecond, 4)
	start := time.Unix(0, 0)
	point := model.Point{X: 20, Y: 30}

	detector.Down(point, start)
	detector.Up(point, start.Add(10*time.Millisecond))
	detector.Down(point, start.Add(100*time.Millisecond))
	events := detector.Up(point, start.Add(120*time.Millisecond))

	if len(events) != 1 || events[0].Kind != model.DoubleClick || events[0].Point != point {
		t.Fatalf("second Up events=%v want one DoubleClick at %v", events, point)
	}
	if events := detector.Tick(start.Add(time.Second)); len(events) != 0 {
		t.Fatalf("Tick after DoubleClick events=%v want none", events)
	}
}

func TestSecondDownInsideWindowLocksDoubleClickPastDeadline(t *testing.T) {
	detector := NewGestureDetector(250*time.Millisecond, 4)
	start := time.Unix(0, 0)
	point := model.Point{X: 20, Y: 30}

	detector.Down(point, start)
	detector.Up(point, start.Add(10*time.Millisecond))
	detector.Down(point, start.Add(250*time.Millisecond))

	if events := detector.Tick(start.Add(260 * time.Millisecond)); len(events) != 0 {
		t.Fatalf("Tick while second button is down events=%v want none", events)
	}
	events := detector.Up(point, start.Add(270*time.Millisecond))
	if len(events) != 1 || events[0].Kind != model.DoubleClick {
		t.Fatalf("second Up after original deadline events=%v want DoubleClick", events)
	}
	if events := detector.Tick(start.Add(time.Second)); len(events) != 0 {
		t.Fatalf("later Tick events=%v want no delayed SingleClick", events)
	}
}

func TestExpiredPendingClickIsEmittedBeforeLaterDrag(t *testing.T) {
	detector := NewGestureDetector(250*time.Millisecond, 4)
	start := time.Unix(0, 0)
	first := model.Point{X: 20, Y: 30}
	second := model.Point{X: 40, Y: 50}

	detector.Down(first, start)
	detector.Up(first, start.Add(10*time.Millisecond))
	events := detector.Down(second, start.Add(300*time.Millisecond))
	if len(events) != 1 || events[0].Kind != model.SingleClick || events[0].Point != first {
		t.Fatalf("late second Down events=%v want pending SingleClick at %v", events, first)
	}

	events = detector.Move(model.Point{X: 45, Y: 50}, start.Add(310*time.Millisecond))
	if len(events) != 1 || events[0].Kind != model.DragStart {
		t.Fatalf("Move events=%v want DragStart", events)
	}
	events = detector.Up(model.Point{X: 45, Y: 50}, start.Add(320*time.Millisecond))
	if len(events) != 1 || events[0].Kind != model.DragEnd {
		t.Fatalf("Up events=%v want DragEnd", events)
	}
	if events := detector.Tick(start.Add(time.Second)); len(events) != 0 {
		t.Fatalf("later Tick events=%v want no extra click", events)
	}
}

func TestMovePastThresholdProducesDragLifecycleWithoutClick(t *testing.T) {
	detector := NewGestureDetector(250*time.Millisecond, 4)
	start := time.Unix(0, 0)
	down := model.Point{X: 10, Y: 10}
	firstMove := model.Point{X: 15, Y: 10}
	secondMove := model.Point{X: 19, Y: 13}

	detector.Down(down, start)
	if events := detector.Move(model.Point{X: 14, Y: 10}, start.Add(10*time.Millisecond)); len(events) != 0 {
		t.Fatalf("Move at threshold events=%v want none", events)
	}
	events := detector.Move(firstMove, start.Add(20*time.Millisecond))
	if len(events) != 1 || events[0].Kind != model.DragStart || events[0].Point != firstMove {
		t.Fatalf("first Move past threshold events=%v want DragStart at %v", events, firstMove)
	}
	events = detector.Move(secondMove, start.Add(30*time.Millisecond))
	if len(events) != 1 || events[0].Kind != model.DragMove || events[0].Point != secondMove {
		t.Fatalf("second drag Move events=%v want DragMove at %v", events, secondMove)
	}
	events = detector.Up(secondMove, start.Add(40*time.Millisecond))
	if len(events) != 1 || events[0].Kind != model.DragEnd || events[0].Point != secondMove {
		t.Fatalf("drag Up events=%v want DragEnd at %v", events, secondMove)
	}
	if events := detector.Tick(start.Add(time.Second)); len(events) != 0 {
		t.Fatalf("Tick after drag events=%v want no click", events)
	}
}

func TestCancelDuringDragEndsDragExactlyOnce(t *testing.T) {
	detector := NewGestureDetector(250*time.Millisecond, 4)
	start := time.Unix(0, 0)
	down := model.Point{X: 10, Y: 10}
	cancel := model.Point{X: 18, Y: 12}

	detector.Down(down, start)
	detector.Move(model.Point{X: 15, Y: 10}, start.Add(10*time.Millisecond))

	events := detector.Cancel(cancel)
	if len(events) != 1 || events[0].Kind != model.DragEnd || events[0].Point != cancel {
		t.Fatalf("Cancel events=%v want one DragEnd at %v", events, cancel)
	}
	if events := detector.Cancel(cancel); len(events) != 0 {
		t.Fatalf("second Cancel events=%v want none", events)
	}
	if events := detector.Up(cancel, start.Add(20*time.Millisecond)); len(events) != 0 {
		t.Fatalf("Up after Cancel events=%v want none", events)
	}
}

func TestDuplicateDownDoesNotResetActivePressOrDrag(t *testing.T) {
	detector := NewGestureDetector(250*time.Millisecond, 4)
	start := time.Unix(0, 0)
	down := model.Point{X: 10, Y: 10}

	detector.Down(down, start)
	detector.Down(model.Point{X: 100, Y: 100}, start.Add(time.Millisecond))
	events := detector.Move(model.Point{X: 15, Y: 10}, start.Add(2*time.Millisecond))
	if len(events) != 1 || events[0].Kind != model.DragStart {
		t.Fatalf("Move after duplicate Down events=%v want DragStart from original press", events)
	}

	detector.Down(model.Point{X: 100, Y: 100}, start.Add(3*time.Millisecond))
	events = detector.Move(model.Point{X: 16, Y: 10}, start.Add(4*time.Millisecond))
	if len(events) != 1 || events[0].Kind != model.DragMove {
		t.Fatalf("Move after duplicate Down during drag events=%v want DragMove", events)
	}
	events = detector.Up(model.Point{X: 16, Y: 10}, start.Add(5*time.Millisecond))
	if len(events) != 1 || events[0].Kind != model.DragEnd {
		t.Fatalf("Up after duplicate Down during drag events=%v want DragEnd", events)
	}
}

func TestMoveUpAndCancelWithoutActivePressAreIgnored(t *testing.T) {
	detector := NewGestureDetector(250*time.Millisecond, 4)
	point := model.Point{X: 10, Y: 10}
	now := time.Unix(0, 0)

	if events := detector.Move(point, now); len(events) != 0 {
		t.Fatalf("Move events=%v want none", events)
	}
	if events := detector.Up(point, now); len(events) != 0 {
		t.Fatalf("Up events=%v want none", events)
	}
	if events := detector.Cancel(point); len(events) != 0 {
		t.Fatalf("Cancel events=%v want none", events)
	}
}
