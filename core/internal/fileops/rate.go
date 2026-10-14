package fileops

import (
	"math"
	"time"
)

const (
	rateWindow = 3 * time.Second
	rateWarmup = 2 * time.Second
)

type rateMeter struct {
	flowing   time.Time
	last      time.Time
	lastBytes int64
	value     float64
}

func (m *rateMeter) restart(now time.Time, bytes int64) {
	m.last = now
	m.lastBytes = bytes
}

func (m *rateMeter) sample(now time.Time, bytes int64) int64 {
	if m.last.IsZero() {
		m.restart(now, bytes)
		return 0
	}
	elapsed := now.Sub(m.last)
	if elapsed <= 0 {
		return m.published(now)
	}
	moved := bytes - m.lastBytes
	m.restart(now, bytes)
	if moved > 0 && m.flowing.IsZero() {
		m.flowing = now.Add(-elapsed)
	}
	if m.flowing.IsZero() {
		return 0
	}

	current := float64(moved) / elapsed.Seconds()
	switch m.value {
	case 0:
		m.value = current
	default:
		weight := 1 - math.Exp(-elapsed.Seconds()/rateWindow.Seconds())
		m.value += weight * (current - m.value)
	}
	return m.published(now)
}

func (m *rateMeter) published(now time.Time) int64 {
	if m.flowing.IsZero() || now.Sub(m.flowing) < rateWarmup {
		return 0
	}
	return int64(m.value)
}
