//go:build windows

package win32

import (
	"fmt"
	"syscall"
	"unsafe"

	"portable-desktop-pet/internal/model"
)

const trayIconID = 1

func (w *Window) addTray() error {
	data := notifyIconData{
		Size:            uint32(unsafe.Sizeof(notifyIconData{})),
		HWnd:            w.hwnd,
		ID:              trayIconID,
		Flags:           nifMessage | nifIcon | nifTip,
		CallbackMessage: wmAppTray,
		Icon:            w.icon,
	}
	copy(data.Tip[:], syscall.StringToUTF16("__PET_TITLE__"))
	ok, _, err := procShellNotifyIconW.Call(nimAdd, uintptr(unsafe.Pointer(&data)))
	if ok == 0 {
		return callError("Shell_NotifyIconW(NIM_ADD)", err)
	}
	w.mu.Lock()
	w.trayAdded = true
	w.mu.Unlock()
	return nil
}

func (w *Window) removeTray() {
	w.mu.Lock()
	if !w.trayAdded {
		w.mu.Unlock()
		return
	}
	w.trayAdded = false
	w.mu.Unlock()
	data := notifyIconData{
		Size: uint32(unsafe.Sizeof(notifyIconData{})),
		HWnd: w.hwnd,
		ID:   trayIconID,
	}
	procShellNotifyIconW.Call(nimDelete, uintptr(unsafe.Pointer(&data)))
}

func (w *Window) ShowMenu(at model.Point) {
	menu, _, _ := procCreatePopupMenu.Call()
	if menu == 0 {
		return
	}
	defer procDestroyMenu.Call(menu)

	topFlag := uintptr(mfString | mfUnchecked)
	if w.Topmost() {
		topFlag = mfString | mfChecked
	}
	appendMenu(menu, topFlag, CommandToggleTopmost, "始终置顶")
	pauseLabel := "暂停活动"
	if w.Paused() {
		pauseLabel = "继续活动"
	}
	appendMenu(menu, mfString, CommandTogglePaused, pauseLabel)
	appendMenu(menu, mfString, CommandResetSize, "恢复默认大小")
	appendMenu(menu, mfString, CommandExit, "退出")

	procSetForegroundWindow.Call(w.hwnd)
	command, _, _ := procTrackPopupMenu.Call(
		menu,
		tpmRightButton|tpmReturnCmd,
		uintptr(at.X), uintptr(at.Y),
		0, w.hwnd, 0,
	)
	if command != 0 {
		w.handleCommand(Command(command))
	}
	procPostMessageW.Call(w.hwnd, 0, 0, 0)
}

func appendMenu(menu, flags uintptr, command Command, label string) {
	text, err := syscall.UTF16PtrFromString(label)
	if err != nil {
		panic(fmt.Sprintf("invalid menu label: %v", err))
	}
	procAppendMenuW.Call(menu, flags, uintptr(command), uintptr(unsafe.Pointer(text)))
}

func (w *Window) handleCommand(command Command) {
	switch command {
	case CommandToggleTopmost:
		w.SetTopmost(!w.Topmost())
	case CommandTogglePaused:
		w.SetPaused(!w.Paused())
	case CommandExit:
		if w.options.OnCommand != nil {
			w.options.OnCommand(command)
		}
		w.Close()
		return
	case CommandResetSize:
		_ = w.ResetSize()
	default:
		return
	}
	if w.options.OnCommand != nil {
		w.options.OnCommand(command)
	}
}
