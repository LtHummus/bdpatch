//go:build !windows

package fsutils

import (
	"fmt"
	"log"
	"os"
	"os/user"
	"syscall"
)

func internalCanModifyTimestamp(path string) bool {
	// on unix systems, we can only set arbitrary last-modified timestamps if 1) we are the owner of the file (write
	// permissions on their own is not enough), or 2) are root
	currentUser, err := user.Current()
	if err != nil {
		log.Printf("[WARNING!] could not get current user: %v", err)
		return false
	}

	if currentUser.Uid == "0" {
		// we are root, so we just own it
		return true
	}

	info, err := os.Stat(path)
	if err != nil {
		log.Printf("[WARNING!] Could not stat %s: %v", path, err)
		return false
	}

	ownerUID := fmt.Sprintf("%d", info.Sys().(*syscall.Stat_t).Uid)

	return currentUser.Uid == ownerUID
}
