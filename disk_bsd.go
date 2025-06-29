//go:build openbsd || netbsd
// +build openbsd netbsd

package main

import "syscall"

// diskUsage for OpenBSD / NetBSD (fields are F_bsize / F_blocks / F_bavail).
func diskUsage(path string) (total, free int64, err error) {
	var st syscall.Statfs_t
	if err = syscall.Statfs(path, &st); err != nil {
		return
	}
	block := int64(st.F_bsize)
	total = int64(st.F_blocks) * block
	free  = int64(st.F_bavail) * block
	return
}

