//go:build linux || darwin || freebsd || dragonfly

package housekeep

import "golang.org/x/sys/unix"

// FreeBytes là số byte người dùng thường còn ghi được trên filesystem chứa path.
func FreeBytes(path string) (uint64, error) {
	var st unix.Statfs_t
	if err := unix.Statfs(path, &st); err != nil {
		return 0, err
	}
	return uint64(st.Bavail) * uint64(st.Bsize), nil
}
