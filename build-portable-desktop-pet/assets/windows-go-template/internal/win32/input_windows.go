//go:build windows

package win32

// Mouse-message translation lives beside windowProc in window_windows.go so
// capture and window movement remain one atomic UI-thread operation. This file
// intentionally marks the Win32 input boundary without duplicating the shared
// input.GestureDetector state machine.
