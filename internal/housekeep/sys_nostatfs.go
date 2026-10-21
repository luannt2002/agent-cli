//go:build !(linux || darwin || freebsd || dragonfly || windows)

package housekeep

import "errors"

// FreeBytes: x/sys không có Statfs dùng chung ở đây (OpenBSD đặt tên trường khác, NetBSD không
// có); người gọi coi như không đo được.
func FreeBytes(string) (uint64, error) { return 0, errors.ErrUnsupported }
