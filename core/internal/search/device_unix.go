//go:build unix

package search

import "syscall"

func deviceOf(path string) (uint64, bool) {
	var st syscall.Stat_t
	if err := syscall.Lstat(path, &st); err != nil {
		return 0, false
	}
	return uint64(st.Dev), true
}
