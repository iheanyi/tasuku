//go:build windows

package v4

import (
	"os"

	"golang.org/x/sys/windows"
)

const lockFileMaxRange = ^uint32(0)

func lockFile(f *os.File) error {
	var overlapped windows.Overlapped
	return windows.LockFileEx(
		windows.Handle(f.Fd()),
		windows.LOCKFILE_EXCLUSIVE_LOCK,
		0,
		lockFileMaxRange,
		lockFileMaxRange,
		&overlapped,
	)
}

func unlockFile(f *os.File) error {
	var overlapped windows.Overlapped
	return windows.UnlockFileEx(
		windows.Handle(f.Fd()),
		0,
		lockFileMaxRange,
		lockFileMaxRange,
		&overlapped,
	)
}
