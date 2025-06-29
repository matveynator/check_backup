//go:build !openbsd && !netbsd
// +build !openbsd,!netbsd

package main

import "golang.org/x/sys/unix"

// diskUsage for Linux, macOS, FreeBSD, etc.
func diskUsage(path string) (total, free int64, err error) {
	var st unix.Statfs_t
	if err = unix.Statfs(path, &st); err != nil {
		return
	}
	block := int64(st.Bsize)          // common field names
	total = int64(st.Blocks) * block
	free  = int64(st.Bavail) * block
	return
}

