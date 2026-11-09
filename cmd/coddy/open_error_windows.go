//go:build windows

package main

import (
	"github.com/lxn/win"
	"golang.org/x/sys/windows"
)

func showOpenError(err error) {
	if err == nil {
		return
	}
	text, textErr := windows.UTF16PtrFromString("Coddy\n\n" + err.Error())
	caption, captionErr := windows.UTF16PtrFromString("Coddy")
	if textErr != nil || captionErr != nil {
		return
	}
	win.MessageBox(0, text, caption, win.MB_OK|win.MB_ICONERROR)
}
