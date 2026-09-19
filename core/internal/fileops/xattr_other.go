//go:build !linux

package fileops

func copyXattrs(_, _ string) error { return nil }
