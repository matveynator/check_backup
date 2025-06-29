//go:build linux || darwin || freebsd || dragonfly || solaris || aix
// +build linux darwin freebsd dragonfly solaris aix

package main

import "syscall"

// diskUsage for platforms where Statfs_t has Bsize / Blocks / Bavail.
func diskUsage(path string) (total, free int64, err error) {
	var st syscall.Statfs_t
	if err = syscall.Statfs(path, &st); err != nil {
		return
	}
	block := int64(st.Bsize)   // bytes per block
	total = int64(st.Blocks) * block
	free  = int64(st.Bavail) * block
	return
}

