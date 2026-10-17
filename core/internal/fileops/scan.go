package fileops

import (
	"errors"
	"io/fs"
	"path/filepath"
	"time"
)

var scanBudget = 2 * time.Second

type scanResult struct {
	files int
	bytes int64
}

var errScanBudget = errors.New("scan budget exceeded")

func (r *runner) scan(paths []string) (int, int64, bool) {
	deadline := time.Now().Add(scanBudget)
	total := scanResult{}
	complete := true

	for _, path := range paths {
		one := scanResult{}
		err := filepath.WalkDir(path, func(_ string, entry fs.DirEntry, err error) error {
			if err != nil {
				return nil
			}
			if r.job.closed() {
				return errScanBudget
			}
			one.files++
			if info, statErr := entry.Info(); statErr == nil && info.Mode().IsRegular() {
				one.bytes += info.Size()
			}
			if time.Now().After(deadline) {
				return errScanBudget
			}
			return nil
		})

		total.files += one.files
		total.bytes += one.bytes
		if err != nil {
			complete = false
			continue
		}
		r.scanned[path] = one
	}
	return total.files, total.bytes, complete
}
