//go:build windows

package win32

import (
	"syscall"
	"unsafe"
)

const (
	wsPopup        = 0x80000000
	wsExTopmost    = 0x00000008
	wsExToolWindow = 0x00000080
	wsExLayered    = 0x00080000

	wmDestroy        = 0x0002
	wmTimer          = 0x0113
	wmCommand        = 0x0111
	wmNCHitTest      = 0x0084
	wmLButtonDown    = 0x0201
	wmLButtonUp      = 0x0202
	wmMouseMove      = 0x0200
	wmRButtonUp      = 0x0205
	wmCaptureChanged = 0x0215
	wmAppTray        = 0x8001
	wmAppClose       = 0x8002

	htTransparent = -1
	htClient      = 1

	ulwAlpha                = 0x00000002
	acSrcOver               = 0x00
	acSrcAlpha              = 0x01
	biRGB                   = 0
	dibRGBColors            = 0
	csHRedraw               = 0x0002
	csVRedraw               = 0x0001
	swpNoSize               = 0x0001
	swpNoMove               = 0x0002
	swpNoActivate           = 0x0010
	spiGetWorkArea          = 0x0030
	monitorDefaultToNearest = 0x00000002
	mfString                = 0x0000
	mfChecked               = 0x0008
	mfUnchecked             = 0x0000
	tpmRightButton          = 0x0002
	tpmReturnCmd            = 0x0100
	nimAdd                  = 0x00000000
	nimDelete               = 0x00000002
	nifMessage              = 0x00000001
	nifIcon                 = 0x00000002
	nifTip                  = 0x00000004
	imageIcon               = 1
	lrDefaultSize           = 0x00000040
	lrShared                = 0x00008000
	idiApplication          = 32512
)

var (
	user32                    = syscall.NewLazyDLL("user32.dll")
	gdi32                     = syscall.NewLazyDLL("gdi32.dll")
	shell32                   = syscall.NewLazyDLL("shell32.dll")
	kernel32                  = syscall.NewLazyDLL("kernel32.dll")
	procRegisterClassExW      = user32.NewProc("RegisterClassExW")
	procCreateWindowExW       = user32.NewProc("CreateWindowExW")
	procDefWindowProcW        = user32.NewProc("DefWindowProcW")
	procDestroyWindow         = user32.NewProc("DestroyWindow")
	procShowWindow            = user32.NewProc("ShowWindow")
	procUpdateWindow          = user32.NewProc("UpdateWindow")
	procGetMessageW           = user32.NewProc("GetMessageW")
	procTranslateMessage      = user32.NewProc("TranslateMessage")
	procDispatchMessageW      = user32.NewProc("DispatchMessageW")
	procPostQuitMessage       = user32.NewProc("PostQuitMessage")
	procGetWindowRect         = user32.NewProc("GetWindowRect")
	procSetWindowPos          = user32.NewProc("SetWindowPos")
	procGetCursorPos          = user32.NewProc("GetCursorPos")
	procGetDoubleClickTime    = user32.NewProc("GetDoubleClickTime")
	procMonitorFromPoint      = user32.NewProc("MonitorFromPoint")
	procGetMonitorInfoW       = user32.NewProc("GetMonitorInfoW")
	procSetTimer              = user32.NewProc("SetTimer")
	procKillTimer             = user32.NewProc("KillTimer")
	procMessageBoxW           = user32.NewProc("MessageBoxW")
	procSetCapture            = user32.NewProc("SetCapture")
	procReleaseCapture        = user32.NewProc("ReleaseCapture")
	procSystemParametersInfoW = user32.NewProc("SystemParametersInfoW")
	procGetDC                 = user32.NewProc("GetDC")
	procReleaseDC             = user32.NewProc("ReleaseDC")
	procUpdateLayeredWindow   = user32.NewProc("UpdateLayeredWindow")
	procCreatePopupMenu       = user32.NewProc("CreatePopupMenu")
	procAppendMenuW           = user32.NewProc("AppendMenuW")
	procTrackPopupMenu        = user32.NewProc("TrackPopupMenu")
	procDestroyMenu           = user32.NewProc("DestroyMenu")
	procSetForegroundWindow   = user32.NewProc("SetForegroundWindow")
	procLoadImageW            = user32.NewProc("LoadImageW")
	procLoadCursorW           = user32.NewProc("LoadCursorW")
	procPostMessageW          = user32.NewProc("PostMessageW")
	procDestroyIcon           = user32.NewProc("DestroyIcon")
	procCreateCompatibleDC    = gdi32.NewProc("CreateCompatibleDC")
	procDeleteDC              = gdi32.NewProc("DeleteDC")
	procCreateDIBSection      = gdi32.NewProc("CreateDIBSection")
	procSelectObject          = gdi32.NewProc("SelectObject")
	procDeleteObject          = gdi32.NewProc("DeleteObject")
	procShellNotifyIconW      = shell32.NewProc("Shell_NotifyIconW")
	procGetModuleHandleW      = kernel32.NewProc("GetModuleHandleW")
)

type point struct {
	X int32
	Y int32
}

type size struct {
	CX int32
	CY int32
}

type rect struct {
	Left   int32
	Top    int32
	Right  int32
	Bottom int32
}

type monitorInfo struct {
	Size    uint32
	Monitor rect
	Work    rect
	Flags   uint32
}

type msg struct {
	HWnd     uintptr
	Message  uint32
	_        uint32
	WParam   uintptr
	LParam   uintptr
	Time     uint32
	Pt       point
	LPrivate uint32
}

type wndClassEx struct {
	Size       uint32
	Style      uint32
	WndProc    uintptr
	ClsExtra   int32
	WndExtra   int32
	Instance   uintptr
	Icon       uintptr
	Cursor     uintptr
	Background uintptr
	MenuName   *uint16
	ClassName  *uint16
	IconSm     uintptr
}

type bitmapInfoHeader struct {
	Size          uint32
	Width         int32
	Height        int32
	Planes        uint16
	BitCount      uint16
	Compression   uint32
	SizeImage     uint32
	XPelsPerMeter int32
	YPelsPerMeter int32
	ClrUsed       uint32
	ClrImportant  uint32
}

type bitmapInfo struct {
	Header bitmapInfoHeader
	Colors [1]uint32
}

type blendFunction struct {
	BlendOp             byte
	BlendFlags          byte
	SourceConstantAlpha byte
	AlphaFormat         byte
}

type notifyIconData struct {
	Size             uint32
	_                uint32
	HWnd             uintptr
	ID               uint32
	Flags            uint32
	CallbackMessage  uint32
	_                uint32
	Icon             uintptr
	Tip              [128]uint16
	State            uint32
	StateMask        uint32
	Info             [256]uint16
	TimeoutOrVersion uint32
	InfoTitle        [64]uint16
	InfoFlags        uint32
	GUIDItem         [16]byte
	BalloonIcon      uintptr
}

// These paired array bounds fail compilation if Go's x64 layout drifts from
// the Win32 ABI expected by the procedures above.
const (
	wndClassExSize       = int(unsafe.Sizeof(wndClassEx{}))
	msgSize              = int(unsafe.Sizeof(msg{}))
	bitmapInfoHeaderSize = int(unsafe.Sizeof(bitmapInfoHeader{}))
	bitmapInfoSize       = int(unsafe.Sizeof(bitmapInfo{}))
	blendFunctionSize    = int(unsafe.Sizeof(blendFunction{}))
	notifyIconDataSize   = int(unsafe.Sizeof(notifyIconData{}))
	monitorInfoSize      = int(unsafe.Sizeof(monitorInfo{}))
)

var (
	_ [80 - wndClassExSize]byte
	_ [wndClassExSize - 80]byte
	_ [48 - msgSize]byte
	_ [msgSize - 48]byte
	_ [40 - bitmapInfoHeaderSize]byte
	_ [bitmapInfoHeaderSize - 40]byte
	_ [44 - bitmapInfoSize]byte
	_ [bitmapInfoSize - 44]byte
	_ [4 - blendFunctionSize]byte
	_ [blendFunctionSize - 4]byte
	_ [976 - notifyIconDataSize]byte
	_ [notifyIconDataSize - 976]byte
	_ [40 - monitorInfoSize]byte
	_ [monitorInfoSize - 40]byte
)

func makeIntResource(id uint16) *uint16 {
	return (*uint16)(unsafe.Pointer(uintptr(id)))
}

func lowWord(value uintptr) uint16 {
	return uint16(value & 0xffff)
}

func signedLowWord(value uintptr) int {
	return int(int16(value & 0xffff))
}

func signedHighWord(value uintptr) int {
	return int(int16((value >> 16) & 0xffff))
}
