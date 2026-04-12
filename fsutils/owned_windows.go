//go:build windows

package fsutils

import (
	"log"

	"golang.org/x/sys/windows"
)

func internalCanModifyTimestamp(path string) bool {
	// ok, i looked in to this. On windows, modifying timestamps just requires FILE_WRITE_ATTRIBUTES, which you
	// will have as long as you have write access to the file. So instead of checking ownership, we can just check
	// to see if we can open the file with write permissions. This also makes the name of this function a little misleading
	// so, uh, we should fix that at some point

	pathPtr, err := windows.UTF16PtrFromString(path)
	if err != nil {
		log.Printf("[WARNING!] Could not get path pointer: %v", err)
		return false
	}

	h, err := windows.CreateFile(pathPtr, windows.FILE_WRITE_ATTRIBUTES, windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE, nil, windows.OPEN_EXISTING, 0, 0)
	if err != nil {
		log.Printf("looks like we don't have write access to modify the file timestamp: %v", err)
		return false
	}
	windows.CloseHandle(h)
	return true
}
