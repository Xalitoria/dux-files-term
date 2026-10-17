package devices

import (
	"sync"
	"time"

	"golang.org/x/sys/unix"
)

const usageTTL = 30 * time.Second

type Usage struct {
	MountPoint string `json:"mountPoint"`
	Total      uint64 `json:"total"`
	Free       uint64 `json:"free"`
	Used       uint64 `json:"used"`
	Code       string `json:"code,omitempty"`
}

type usageCache struct {
	now func() time.Time

	mu      sync.Mutex
	entries map[string]usageEntry
}

type usageEntry struct {
	usage Usage
	at    time.Time
}

func newUsageCache() *usageCache {
	return &usageCache{now: time.Now, entries: map[string]usageEntry{}}
}

func (c *usageCache) get(mountPoint string, force bool) Usage {
	c.mu.Lock()
	cached, ok := c.entries[mountPoint]
	c.mu.Unlock()

	if ok && !force && c.now().Sub(cached.at) < usageTTL {
		return cached.usage
	}

	fresh := statfs(mountPoint)
	c.mu.Lock()
	c.entries[mountPoint] = usageEntry{usage: fresh, at: c.now()}
	c.mu.Unlock()
	return fresh
}

func (c *usageCache) forget(mountPoint string) {
	c.mu.Lock()
	delete(c.entries, mountPoint)
	c.mu.Unlock()
}

func statfs(mountPoint string) Usage {
	var stat unix.Statfs_t
	if err := unix.Statfs(mountPoint, &stat); err != nil {
		return Usage{MountPoint: mountPoint, Code: codeForErrno(err)}
	}

	block := uint64(stat.Bsize)
	total := stat.Blocks * block
	free := stat.Bavail * block
	return Usage{
		MountPoint: mountPoint,
		Total:      total,
		Free:       free,
		Used:       total - stat.Bfree*block,
	}
}

func codeForErrno(err error) string {
	switch err {
	case unix.ENOENT:
		return "ENOENT"
	case unix.EACCES, unix.EPERM:
		return "EACCES"
	case unix.ENOTDIR:
		return "ENOTDIR"
	default:
		return "EIO"
	}
}
