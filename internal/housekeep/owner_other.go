//go:build !unix

package housekeep

import "io/fs"

// fileUID: hệ điều hành không có uid (Windows...).
func fileUID(fs.FileInfo) (int, bool) { return 0, false }
