//go:build windows

package win32

import (
	"errors"
	"fmt"
	"image"
	"runtime"
	"sync"
	"syscall"
	"time"
	"unsafe"

	"portable-desktop-pet/internal/input"
	"portable-desktop-pet/internal/model"
	"portable-desktop-pet/internal/screen"
)

type Options struct {
	Title        string
	InitialSize  model.Size
	Gesture      *input.GestureDetector
	OnEvents     func([]model.Event)
	OnCommand    func(Command)
	WorkArea     func(model.Point) model.Rect
	TickInterval time.Duration
	OnTick       func(time.Time)
}

type Window struct {
	hwnd           uintptr
	instance       uintptr
	icon           uintptr
	iconOwned      bool
	options        Options
	mu             sync.RWMutex
	frame          preparedFrame
	lastFrame      image.Image
	position       model.Point
	defaultSize    model.Size
	renderSize     model.Size
	topmost        bool
	paused         bool
	closed         bool
	closeRequested bool
	threadLocked   bool
	trayAdded      bool
	timerID        uintptr
	dragCursor     point
	dragOrigin     point
	captured       bool
}

var (
	windowsMu sync.RWMutex
	windows   = make(map[uintptr]*Window)
	wndProc   = syscall.NewCallback(windowProc)
	classOnce sync.Once
	classErr  error
	className = syscall.StringToUTF16Ptr("PortableDesktopPetWindow")
)

func NewWindow(options Options) (*Window, error) {
	// A Win32 window and its message loop are OS-thread-affine. NewWindow
	// locks this goroutine to the creation thread; the caller must invoke Run
	// on the same goroutine. Run releases the lock when the loop terminates.
	runtime.LockOSThread()
	keepThreadLocked := false
	defer func() {
		if !keepThreadLocked {
			runtime.UnlockOSThread()
		}
	}()

	if options.Title == "" {
		options.Title = "__PET_TITLE__"
	}
	if options.InitialSize.W <= 0 {
		options.InitialSize.W = 190
	}
	if options.InitialSize.H <= 0 {
		options.InitialSize.H = 190
	}
	if options.Gesture == nil {
		options.Gesture = input.NewGestureDetector(500*time.Millisecond, 4)
	}

	instance, _, err := procGetModuleHandleW.Call(0)
	if instance == 0 {
		return nil, callError("GetModuleHandleW", err)
	}
	classOnce.Do(func() {
		classErr = registerWindowClass(instance)
	})
	if classErr != nil {
		return nil, classErr
	}

	var work rect
	ok, _, err := procSystemParametersInfoW.Call(spiGetWorkArea, 0, uintptr(unsafe.Pointer(&work)), 0)
	if ok == 0 {
		return nil, callError("SystemParametersInfoW", err)
	}
	x := int(work.Right) - options.InitialSize.W - 20
	y := int(work.Bottom) - options.InitialSize.H - 20
	if x < int(work.Left) {
		x = int(work.Left)
	}
	if y < int(work.Top) {
		y = int(work.Top)
	}

	title, errText := syscall.UTF16PtrFromString(options.Title)
	if errText != nil {
		return nil, errText
	}
	hwnd, _, err := procCreateWindowExW.Call(
		wsExLayered|wsExToolWindow|wsExTopmost,
		uintptr(unsafe.Pointer(className)),
		uintptr(unsafe.Pointer(title)),
		wsPopup,
		uintptr(x), uintptr(y),
		uintptr(options.InitialSize.W), uintptr(options.InitialSize.H),
		0, 0, instance, 0,
	)
	if hwnd == 0 {
		return nil, callError("CreateWindowExW", err)
	}

	w := &Window{
		hwnd:         hwnd,
		instance:     instance,
		options:      options,
		position:     model.Point{X: x, Y: y},
		defaultSize:  options.InitialSize,
		renderSize:   options.InitialSize,
		topmost:      true,
		threadLocked: true,
	}
	w.icon, _, _ = procLoadImageW.Call(instance, uintptr(unsafe.Pointer(makeIntResource(1))), imageIcon, 0, 0, lrDefaultSize)
	w.iconOwned = w.icon != 0
	if w.icon == 0 {
		w.icon, _, _ = procLoadImageW.Call(0, uintptr(unsafe.Pointer(makeIntResource(idiApplication))), imageIcon, 0, 0, lrDefaultSize|lrShared)
		w.iconOwned = false
	}
	windowsMu.Lock()
	windows[hwnd] = w
	windowsMu.Unlock()
	if err := w.addTray(); err != nil {
		windowsMu.Lock()
		delete(windows, hwnd)
		windowsMu.Unlock()
		procDestroyWindow.Call(hwnd)
		w.cleanupIcon()
		return nil, err
	}
	if options.TickInterval > 0 {
		milliseconds := options.TickInterval.Milliseconds()
		if milliseconds < 1 {
			milliseconds = 1
		}
		timerID, _, err := procSetTimer.Call(hwnd, 1, uintptr(milliseconds), 0)
		if timerID == 0 {
			w.removeTray()
			windowsMu.Lock()
			delete(windows, hwnd)
			windowsMu.Unlock()
			procDestroyWindow.Call(hwnd)
			w.cleanupIcon()
			return nil, callError("SetTimer", err)
		}
		w.timerID = timerID
	}
	procShowWindow.Call(hwnd, 5)
	procUpdateWindow.Call(hwnd)
	keepThreadLocked = true
	return w, nil
}

func registerWindowClass(instance uintptr) error {
	cursor, _, _ := procLoadCursorW.Call(0, uintptr(unsafe.Pointer(makeIntResource(32512))))
	wc := wndClassEx{
		Size:      uint32(unsafe.Sizeof(wndClassEx{})),
		Style:     csHRedraw | csVRedraw,
		WndProc:   wndProc,
		Instance:  instance,
		Cursor:    cursor,
		ClassName: className,
	}
	atom, _, err := procRegisterClassExW.Call(uintptr(unsafe.Pointer(&wc)))
	if atom == 0 {
		return callError("RegisterClassExW", err)
	}
	return nil
}

func (w *Window) Run() int {
	defer func() {
		w.mu.Lock()
		locked := w.threadLocked
		if locked {
			w.threadLocked = false
		}
		w.mu.Unlock()
		if locked {
			runtime.UnlockOSThread()
		}
	}()
	var message msg
	for {
		result, _, _ := procGetMessageW.Call(uintptr(unsafe.Pointer(&message)), 0, 0, 0)
		if int32(result) == -1 {
			w.destroyOnUIThread()
			return 1
		}
		if result == 0 {
			return int(message.WParam)
		}
		procTranslateMessage.Call(uintptr(unsafe.Pointer(&message)))
		procDispatchMessageW.Call(uintptr(unsafe.Pointer(&message)))
	}
}

func (w *Window) Render(frame image.Image, pos model.Point) error {
	w.mu.RLock()
	renderSize := w.renderSize
	w.mu.RUnlock()
	prepared := prepareFrame(scaleFrame(frame, renderSize))
	if prepared.Width == 0 || prepared.Height == 0 {
		return errors.New("cannot render an empty frame")
	}

	screenDC, _, err := procGetDC.Call(0)
	if screenDC == 0 {
		return callError("GetDC", err)
	}
	defer procReleaseDC.Call(0, screenDC)
	memoryDC, _, err := procCreateCompatibleDC.Call(screenDC)
	if memoryDC == 0 {
		return callError("CreateCompatibleDC", err)
	}
	defer procDeleteDC.Call(memoryDC)

	info := bitmapInfo{Header: bitmapInfoHeader{
		Size:        uint32(unsafe.Sizeof(bitmapInfoHeader{})),
		Width:       int32(prepared.Width),
		Height:      -int32(prepared.Height),
		Planes:      1,
		BitCount:    32,
		Compression: biRGB,
		SizeImage:   uint32(len(prepared.Pixels)),
	}}
	var bits uintptr
	bitmap, _, err := procCreateDIBSection.Call(
		memoryDC,
		uintptr(unsafe.Pointer(&info)),
		dibRGBColors,
		uintptr(unsafe.Pointer(&bits)),
		0, 0,
	)
	if bitmap == 0 || bits == 0 {
		return callError("CreateDIBSection", err)
	}
	defer procDeleteObject.Call(bitmap)
	copy(unsafe.Slice((*byte)(unsafe.Pointer(bits)), len(prepared.Pixels)), prepared.Pixels)
	old, _, _ := procSelectObject.Call(memoryDC, bitmap)
	defer procSelectObject.Call(memoryDC, old)

	destination := point{X: int32(pos.X), Y: int32(pos.Y)}
	dimensions := size{CX: int32(prepared.Width), CY: int32(prepared.Height)}
	source := point{}
	blend := blendFunction{BlendOp: acSrcOver, SourceConstantAlpha: 255, AlphaFormat: acSrcAlpha}
	ok, _, err := procUpdateLayeredWindow.Call(
		w.hwnd,
		screenDC,
		uintptr(unsafe.Pointer(&destination)),
		uintptr(unsafe.Pointer(&dimensions)),
		memoryDC,
		uintptr(unsafe.Pointer(&source)),
		0,
		uintptr(unsafe.Pointer(&blend)),
		ulwAlpha,
	)
	if ok == 0 {
		return callError("UpdateLayeredWindow", err)
	}
	w.mu.Lock()
	w.frame = prepared
	w.lastFrame = frame
	w.position = pos
	w.mu.Unlock()
	return nil
}

func (w *Window) SetTopmost(enabled bool) {
	insertAfter := ^uintptr(0) // HWND_TOPMOST (-1)
	if !enabled {
		insertAfter = ^uintptr(1) // HWND_NOTOPMOST (-2)
	}
	procSetWindowPos.Call(w.hwnd, insertAfter, 0, 0, 0, 0, swpNoMove|swpNoSize|swpNoActivate)
	w.mu.Lock()
	w.topmost = enabled
	w.mu.Unlock()
}

func (w *Window) SetPaused(paused bool) {
	w.mu.Lock()
	w.paused = paused
	w.mu.Unlock()
}

func (w *Window) Paused() bool {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.paused
}

func (w *Window) Topmost() bool {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.topmost
}

func (w *Window) Position() model.Point {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.position
}

func (w *Window) RenderSize() model.Size {
	w.mu.RLock()
	defer w.mu.RUnlock()
	if w.frame.Width > 0 && w.frame.Height > 0 {
		return model.Size{W: w.frame.Width, H: w.frame.Height}
	}
	return w.renderSize
}

func CursorPosition() (model.Point, bool) {
	var cursor point
	if ok, _, _ := procGetCursorPos.Call(uintptr(unsafe.Pointer(&cursor))); ok == 0 {
		return model.Point{}, false
	}
	return model.Point{X: int(cursor.X), Y: int(cursor.Y)}, true
}

func DoubleClickInterval() time.Duration {
	milliseconds, _, _ := procGetDoubleClickTime.Call()
	if milliseconds == 0 {
		return 500 * time.Millisecond
	}
	return time.Duration(milliseconds) * time.Millisecond
}

func WorkAreaAt(at model.Point) (model.Rect, bool) {
	nativePoint := point{X: int32(at.X), Y: int32(at.Y)}
	packedPoint := uintptr(uint32(nativePoint.X)) | uintptr(uint64(uint32(nativePoint.Y))<<32)
	monitor, _, _ := procMonitorFromPoint.Call(
		packedPoint,
		monitorDefaultToNearest,
	)
	if monitor != 0 {
		info := monitorInfo{Size: uint32(unsafe.Sizeof(monitorInfo{}))}
		if ok, _, _ := procGetMonitorInfoW.Call(monitor, uintptr(unsafe.Pointer(&info))); ok != 0 {
			return model.Rect{
				Left: int(info.Work.Left), Top: int(info.Work.Top),
				Right: int(info.Work.Right), Bottom: int(info.Work.Bottom),
			}, true
		}
	}

	var primary rect
	if ok, _, _ := procSystemParametersInfoW.Call(spiGetWorkArea, 0, uintptr(unsafe.Pointer(&primary)), 0); ok == 0 {
		return model.Rect{}, false
	}
	return model.Rect{
		Left: int(primary.Left), Top: int(primary.Top),
		Right: int(primary.Right), Bottom: int(primary.Bottom),
	}, true
}

func ShowError(title, message string) {
	titleText, titleErr := syscall.UTF16PtrFromString(title)
	messageText, messageErr := syscall.UTF16PtrFromString(message)
	if titleErr != nil || messageErr != nil {
		return
	}
	const mbOKIconError = 0x00000000 | 0x00000010
	procMessageBoxW.Call(
		0,
		uintptr(unsafe.Pointer(messageText)),
		uintptr(unsafe.Pointer(titleText)),
		mbOKIconError,
	)
}

func (w *Window) SetRenderSize(size model.Size) error {
	if size.W <= 0 || size.H <= 0 {
		return fmt.Errorf("invalid render size %dx%d", size.W, size.H)
	}
	w.mu.Lock()
	w.renderSize = size
	frame := w.lastFrame
	position := w.position
	w.mu.Unlock()
	if frame != nil {
		return w.Render(frame, position)
	}
	return nil
}

func (w *Window) ResetSize() error {
	return w.SetRenderSize(w.defaultSize)
}

func (w *Window) Close() {
	w.mu.Lock()
	if w.closed || w.closeRequested {
		w.mu.Unlock()
		return
	}
	w.closeRequested = true
	w.mu.Unlock()
	ok, _, _ := procPostMessageW.Call(w.hwnd, wmAppClose, 0, 0)
	if ok == 0 {
		w.mu.Lock()
		if !w.closed {
			w.closeRequested = false
		}
		w.mu.Unlock()
	}
}

func (w *Window) dispatch(events []model.Event) {
	if len(events) != 0 && w.options.OnEvents != nil {
		w.options.OnEvents(events)
	}
}

func (w *Window) hit(screenX, screenY int) bool {
	var bounds rect
	ok, _, _ := procGetWindowRect.Call(w.hwnd, uintptr(unsafe.Pointer(&bounds)))
	if ok == 0 {
		return false
	}
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.frame.Mask.HitWithin(screenX-int(bounds.Left), screenY-int(bounds.Top), 6)
}

func (w *Window) beginPotentialDrag() {
	procGetCursorPos.Call(uintptr(unsafe.Pointer(&w.dragCursor)))
	var bounds rect
	if ok, _, _ := procGetWindowRect.Call(w.hwnd, uintptr(unsafe.Pointer(&bounds))); ok != 0 {
		w.dragOrigin = point{X: bounds.Left, Y: bounds.Top}
	}
}

func (w *Window) applyDrag(events []model.Event) {
	for _, event := range events {
		switch event.Kind {
		case model.DragStart:
			procSetCapture.Call(w.hwnd)
			w.captured = true
			w.moveToCursor()
		case model.DragMove:
			w.moveToCursor()
		case model.DragEnd:
			if w.captured {
				procReleaseCapture.Call()
				w.captured = false
			}
			w.clampAfterDrag()
		}
	}
}

func (w *Window) clampAfterDrag() {
	var bounds rect
	if ok, _, _ := procGetWindowRect.Call(w.hwnd, uintptr(unsafe.Pointer(&bounds))); ok == 0 {
		return
	}
	position := model.Point{X: int(bounds.Left), Y: int(bounds.Top)}
	windowSize := model.Size{W: int(bounds.Right - bounds.Left), H: int(bounds.Bottom - bounds.Top)}
	var work model.Rect
	if w.options.WorkArea != nil {
		work = w.options.WorkArea(position)
	} else {
		var primary rect
		if ok, _, _ := procSystemParametersInfoW.Call(spiGetWorkArea, 0, uintptr(unsafe.Pointer(&primary)), 0); ok == 0 {
			return
		}
		work = model.Rect{Left: int(primary.Left), Top: int(primary.Top), Right: int(primary.Right), Bottom: int(primary.Bottom)}
	}
	clamped := screen.ClampWindow(position, windowSize, work)
	procSetWindowPos.Call(w.hwnd, 0, uintptr(clamped.X), uintptr(clamped.Y), 0, 0, swpNoSize|swpNoActivate)
	w.mu.Lock()
	w.position = clamped
	w.mu.Unlock()
}

func (w *Window) moveToCursor() {
	var current point
	if ok, _, _ := procGetCursorPos.Call(uintptr(unsafe.Pointer(&current))); ok == 0 {
		return
	}
	x := w.dragOrigin.X + current.X - w.dragCursor.X
	y := w.dragOrigin.Y + current.Y - w.dragCursor.Y
	procSetWindowPos.Call(w.hwnd, 0, uintptr(x), uintptr(y), 0, 0, swpNoSize|swpNoActivate)
	w.mu.Lock()
	w.position = model.Point{X: int(x), Y: int(y)}
	w.mu.Unlock()
}

func windowProc(hwnd uintptr, message uint32, wParam, lParam uintptr) uintptr {
	windowsMu.RLock()
	w := windows[hwnd]
	windowsMu.RUnlock()
	if w == nil {
		result, _, _ := procDefWindowProcW.Call(hwnd, uintptr(message), wParam, lParam)
		return result
	}

	now := time.Now()
	clientPoint := model.Point{X: signedLowWord(lParam), Y: signedHighWord(lParam)}
	switch message {
	case wmNCHitTest:
		if w.hit(signedLowWord(lParam), signedHighWord(lParam)) {
			return htClient
		}
		return uintptr(^uint(0)) // HTTRANSPARENT (-1)
	case wmLButtonDown:
		w.beginPotentialDrag()
		w.dispatch(w.options.Gesture.Down(clientPoint, now))
		return 0
	case wmMouseMove:
		events := w.options.Gesture.Move(clientPoint, now)
		w.applyDrag(events)
		w.dispatch(events)
		return 0
	case wmLButtonUp:
		events := w.options.Gesture.Up(clientPoint, now)
		w.applyDrag(events)
		w.dispatch(events)
		return 0
	case wmCaptureChanged:
		w.captured = false
		events := w.options.Gesture.Cancel(w.cursorClientPoint())
		w.clampAfterDrag()
		w.dispatch(events)
		return 0
	case wmAppClose:
		w.destroyOnUIThread()
		return 0
	case wmTimer:
		if w.timerID != 0 && wParam == w.timerID {
			w.dispatch(w.options.Gesture.Tick(now))
			if w.options.OnTick != nil {
				w.options.OnTick(now)
			}
		}
		return 0
	case wmRButtonUp:
		w.dispatch([]model.Event{{Kind: model.ContextMenu, Point: clientPoint}})
		var cursor point
		procGetCursorPos.Call(uintptr(unsafe.Pointer(&cursor)))
		w.ShowMenu(model.Point{X: int(cursor.X), Y: int(cursor.Y)})
		return 0
	case wmAppTray:
		if lowWord(lParam) == wmRButtonUp {
			var cursor point
			procGetCursorPos.Call(uintptr(unsafe.Pointer(&cursor)))
			w.ShowMenu(model.Point{X: int(cursor.X), Y: int(cursor.Y)})
		}
		return 0
	case wmCommand:
		w.handleCommand(Command(lowWord(wParam)))
		return 0
	case wmDestroy:
		if w.timerID != 0 {
			procKillTimer.Call(hwnd, w.timerID)
			w.timerID = 0
		}
		w.removeTray()
		w.cleanupIcon()
		w.mu.Lock()
		w.closed = true
		w.closeRequested = false
		w.mu.Unlock()
		windowsMu.Lock()
		delete(windows, hwnd)
		windowsMu.Unlock()
		procPostQuitMessage.Call(0)
		return 0
	}
	result, _, _ := procDefWindowProcW.Call(hwnd, uintptr(message), wParam, lParam)
	return result
}

func (w *Window) destroyOnUIThread() {
	w.removeTray()
	ok, _, _ := procDestroyWindow.Call(w.hwnd)
	if ok == 0 {
		w.mu.Lock()
		if !w.closed {
			w.closeRequested = false
		}
		w.mu.Unlock()
	}
}

func (w *Window) cleanupIcon() {
	w.mu.Lock()
	icon := w.icon
	owned := w.iconOwned
	w.icon = 0
	w.iconOwned = false
	w.mu.Unlock()
	if owned && icon != 0 {
		procDestroyIcon.Call(icon)
	}
}

func (w *Window) cursorClientPoint() model.Point {
	var cursor point
	var bounds rect
	if ok, _, _ := procGetCursorPos.Call(uintptr(unsafe.Pointer(&cursor))); ok == 0 {
		return model.Point{}
	}
	if ok, _, _ := procGetWindowRect.Call(w.hwnd, uintptr(unsafe.Pointer(&bounds))); ok == 0 {
		return model.Point{}
	}
	return model.Point{X: int(cursor.X - bounds.Left), Y: int(cursor.Y - bounds.Top)}
}

func callError(name string, err error) error {
	if errno, ok := err.(syscall.Errno); ok && errno == 0 {
		return fmt.Errorf("%s failed", name)
	}
	return fmt.Errorf("%s failed: %w", name, err)
}
