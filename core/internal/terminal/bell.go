package terminal

import (
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"
)

// Programs that ring in a loop get one sound per cooldown.
const bellCooldown = 250 * time.Millisecond

var bellPlayer = sync.OnceValue(findBellPlayer)

// The emulator calls this while the session lock is held.
func (s *Session) ring() {
	s.rang = true
	s.scheduleLocked()
	if !s.bellAudio || time.Since(s.lastBell) < bellCooldown {
		return
	}
	s.lastBell = time.Now()
	playBell()
}

func playBell() {
	argv := bellPlayer()
	if argv == nil {
		return
	}
	cmd := exec.Command(argv[0], argv[1:]...)
	if err := cmd.Start(); err != nil {
		return
	}
	go func() { _ = cmd.Wait() }()
}

// The desktop's bell sound through whichever player is installed; libcanberra knows the sound theme itself.
func findBellPlayer() []string {
	if path, err := exec.LookPath("canberra-gtk-play"); err == nil {
		return []string{path, "-i", "bell"}
	}
	sound := bellSound()
	if sound == "" {
		return nil
	}
	for _, name := range []string{"pw-play", "paplay"} {
		if path, err := exec.LookPath(name); err == nil {
			return []string{path, sound}
		}
	}
	return nil
}

func bellSound() string {
	dirs := append(filepath.SplitList(os.Getenv("XDG_DATA_DIRS")), "/usr/local/share", "/usr/share")
	for _, dir := range dirs {
		path := filepath.Join(dir, "sounds", "freedesktop", "stereo", "bell.oga")
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}
	return ""
}
