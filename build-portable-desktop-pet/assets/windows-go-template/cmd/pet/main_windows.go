//go:build windows

package main

import (
	"os"

	"portable-desktop-pet/internal/app"
)

func main() {
	os.Exit(app.Run())
}
