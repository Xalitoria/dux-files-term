//go:build !unix

package search

func deviceOf(string) (uint64, bool) { return 0, false }
