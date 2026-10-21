//go:build windows

package housekeep

import "golang.org/x/sys/windows"

// FreeBytes là số byte người dùng còn ghi được trên ổ chứa path.
func FreeBytes(path string) (uint64, error) {
	p, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return 0, err
	}
	var free uint64
	err = windows.GetDiskFreeSpaceEx(p, &free, nil, nil)
	return free, err
}
