//go:build (openbsd || netbsd) && !cgo
// +build openbsd netbsd
// +build !cgo

package main

import (
	"bytes"
	"errors"
	"os/exec"
	"strconv"
	"strings"
)

// diskUsage via `df -k` when CGO is off on *BSD.
func diskUsage(path string) (total, free int64, err error) {
	out, err := exec.Command("df", "-k", path).Output()
	if err != nil {
		return 0, 0, err
	}
	// skip header line, split by fields
	lines := bytes.Split(out, []byte{'\n'})
	if len(lines) < 2 {
		return 0, 0, errors.New("df output parse error")
	}
	fields := strings.Fields(string(lines[1]))
	if len(fields) < 5 {
		return 0, 0, errors.New("df output parse error")
	}
	// df -k shows blocks in KiB
	totalKB, _ := strconv.ParseInt(fields[1], 10, 64)
	freeKB, _ := strconv.ParseInt(fields[3], 10, 64)
	return totalKB * 1024, freeKB * 1024, nil
}
