//go:build windows

package app

import (
	"fmt"
	"time"

	"portable-desktop-pet/internal/animation"
	"portable-desktop-pet/internal/assets"
	"portable-desktop-pet/internal/behavior"
	"portable-desktop-pet/internal/input"
	"portable-desktop-pet/internal/model"
	"portable-desktop-pet/internal/win32"
)

const (
	tickInterval     = 90 * time.Millisecond
	defaultWidth     = 190
	defaultHeight    = 190
	gazeDeadRadius   = 24
	gazeNearbyRadius = 380
)

type desktopPet struct {
	window     *win32.Window
	bundle     *assets.Bundle
	animation  *animation.Controller
	behavior   *behavior.Controller
	state      model.State
	target     model.Point
	hasTarget  bool
	nextChoice time.Time

	lastRender    renderSnapshot
	hasLastRender bool
	fatal         error
}

func Run() int {
	bundle, err := assets.Load()
	if err != nil {
		win32.ShowError("__PET_TITLE__", fmt.Sprintf("资源加载失败：%v", err))
		return 1
	}

	now := time.Now()
	pet := &desktopPet{
		bundle:     bundle,
		animation:  animation.NewController(bundle.ClipMap()),
		behavior:   behavior.New(now.UnixNano()),
		state:      model.Idle,
		nextChoice: now.Add(stateHoldDuration(model.Idle)),
	}
	gesture := input.NewGestureDetector(win32.DoubleClickInterval(), 4)

	window, err := win32.NewWindow(win32.Options{
		Title:        "__PET_TITLE__",
		InitialSize:  model.Size{W: defaultWidth, H: defaultHeight},
		Gesture:      gesture,
		TickInterval: tickInterval,
		OnEvents: func(events []model.Event) {
			pet.handleEvents(events, time.Now())
		},
		OnCommand: pet.handleCommand,
		OnTick: func(tickTime time.Time) {
			pet.tick(tickTime)
		},
		WorkArea: func(point model.Point) model.Rect {
			work, _ := win32.WorkAreaAt(point)
			return work
		},
	})
	if err != nil {
		win32.ShowError("__PET_TITLE__", fmt.Sprintf("窗口创建失败：%v", err))
		return 1
	}
	pet.window = window
	pet.animation.Play(model.Idle, now)
	pet.behavior.SetPosition(window.Position(), window.RenderSize())
	pet.render(now, true)
	if pet.fatal != nil {
		win32.ShowError("__PET_TITLE__", pet.fatal.Error())
		window.Close()
		_ = window.Run()
		return 1
	}

	exitCode := window.Run()
	if pet.fatal != nil {
		win32.ShowError("__PET_TITLE__", pet.fatal.Error())
		return 1
	}
	if exitCode != 0 {
		win32.ShowError("__PET_TITLE__", "Windows 消息循环异常结束。")
		return 1
	}
	return 0
}

func (p *desktopPet) handleEvents(events []model.Event, now time.Time) {
	if p.window == nil || p.fatal != nil {
		return
	}
	for _, event := range events {
		if event.Kind == model.ContextMenu {
			continue
		}
		previous := p.state
		p.state = p.behavior.Request(event, now)
		p.hasTarget = false
		p.nextChoice = now.Add(stateHoldDuration(p.state))

		if event.Kind == model.DragEnd {
			p.behavior.SetPosition(p.window.Position(), p.window.RenderSize())
			p.nextChoice = now
		}
		if p.state != previous {
			p.animation.Play(p.state, now)
		}
	}
}

func (p *desktopPet) handleCommand(command win32.Command) {
	if p.window == nil || p.fatal != nil {
		return
	}
	now := time.Now()
	switch command {
	case win32.CommandTogglePaused:
		paused := p.window.Paused()
		p.behavior.SetPaused(paused)
		if paused {
			p.setState(model.Idle, now)
			p.hasTarget = false
		}
		p.nextChoice = now
	case win32.CommandResetSize:
		p.behavior.SetPosition(p.window.Position(), p.window.RenderSize())
		p.hasTarget = false
		p.nextChoice = now
	}
	p.render(now, false)
}

func (p *desktopPet) tick(now time.Time) {
	if p.window == nil || p.fatal != nil {
		return
	}

	position := p.window.Position()
	size := p.window.RenderSize()
	work, ok := win32.WorkAreaAt(position)
	if !ok {
		p.fail(fmt.Errorf("无法读取显示器工作区"))
		return
	}
	p.behavior.UpdatePosition(position, size)

	if !now.Before(p.nextChoice) && p.state != model.Drag {
		decision := p.behavior.NextAuto(now, work)
		p.setState(decision.State, now)
		p.target = decision.Target
		p.hasTarget = decision.HasTarget
		hold := stateHoldDuration(p.state)
		if hold <= 0 {
			hold = tickInterval
		}
		p.nextChoice = now.Add(hold)
	}

	if p.hasTarget && isMovementState(p.state) {
		next, reached, corrected := stepMotion(position, p.target, p.state, size, work)
		position = next
		if corrected {
			p.behavior.SetPosition(position, size)
		} else {
			p.behavior.UpdatePosition(position, size)
		}
		if reached {
			p.hasTarget = false
			p.nextChoice = now
		}
	}
	p.renderAt(now, position, false)
}

func (p *desktopPet) setState(state model.State, now time.Time) {
	if state == p.state {
		return
	}
	p.state = state
	p.animation.Play(state, now)
}

func (p *desktopPet) render(now time.Time, force bool) {
	if p.window == nil {
		return
	}
	p.renderAt(now, p.window.Position(), force)
}

func (p *desktopPet) renderAt(now time.Time, position model.Point, force bool) {
	frameRef, _ := p.animation.Frame(now)
	size := p.window.RenderSize()
	if p.state == model.Idle {
		if cursor, ok := win32.CursorPosition(); ok {
			center := model.Point{X: position.X + size.W/2, Y: position.Y + size.H/2}
			dx, dy := cursor.X-center.X, cursor.Y-center.Y
			if dx*dx+dy*dy <= gazeNearbyRadius*gazeNearbyRadius {
				if gaze, active := gazeFrame(center, cursor, gazeDeadRadius); active {
					frameRef = gaze
				}
			}
		}
	}

	next := renderSnapshot{
		Frame: frameRef, Position: position,
		Paused: p.window.Paused(), Topmost: p.window.Topmost(),
	}
	if !force && p.hasLastRender && !needsRender(p.lastRender, next) {
		return
	}
	frame, err := p.bundle.Standard.Frame(frameRef.Row, frameRef.Col)
	if err != nil {
		p.fail(fmt.Errorf("读取动画帧失败：%w", err))
		return
	}
	if err := p.window.Render(frame, position); err != nil {
		p.fail(fmt.Errorf("绘制角色失败：%w", err))
		return
	}
	p.lastRender = next
	p.hasLastRender = true
}

func (p *desktopPet) fail(err error) {
	if err == nil || p.fatal != nil {
		return
	}
	p.fatal = err
	if p.window != nil {
		p.window.Close()
	}
}
