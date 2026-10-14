package fileops

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

const megabyte = 1_000_000

func feed(m *rateMeter, start time.Time, from, to time.Duration, step time.Duration, bytesPerSec int64, bytes *int64) (time.Time, int64) {
	var published int64
	now := start.Add(from)
	for at := from + step; at <= to; at += step {
		now = start.Add(at)
		*bytes += bytesPerSec * int64(step) / int64(time.Second)
		published = m.sample(now, *bytes)
	}
	return now, published
}

func TestRateStaysUnknownDuringWarmup(t *testing.T) {
	var m rateMeter
	var bytes int64
	start := time.Unix(0, 0)
	m.sample(start, 0)

	_, published := feed(&m, start, 0, rateWarmup-tickInterval, tickInterval, 10*megabyte, &bytes)
	assert.Zero(t, published)

	_, published = feed(&m, start, rateWarmup-tickInterval, rateWarmup, tickInterval, 10*megabyte, &bytes)
	assert.InDelta(t, 10*megabyte, published, megabyte/100)
}

func TestWarmupCountsFromTheFirstByte(t *testing.T) {
	var m rateMeter
	var bytes int64
	start := time.Unix(0, 0)
	m.sample(start, 0)

	feed(&m, start, 0, 5*time.Second, tickInterval, 0, &bytes)
	_, published := feed(&m, start, 5*time.Second, 6*time.Second, tickInterval, 10*megabyte, &bytes)
	assert.Zero(t, published, "a long scan before the first byte is not warmup")
}

func TestRateSmoothsASpike(t *testing.T) {
	var m rateMeter
	var bytes int64
	start := time.Unix(0, 0)
	m.sample(start, 0)

	_, steady := feed(&m, start, 0, 10*time.Second, tickInterval, 10*megabyte, &bytes)
	_, spiked := feed(&m, start, 10*time.Second, 10*time.Second+tickInterval, tickInterval, 100*megabyte, &bytes)

	assert.InDelta(t, 10*megabyte, steady, megabyte/100)
	assert.Greater(t, spiked, steady)
	assert.Less(t, spiked, int64(30*megabyte), "one fast tick moves the rate, it does not replace it")
}

func TestRateFollowsASustainedChange(t *testing.T) {
	var m rateMeter
	var bytes int64
	start := time.Unix(0, 0)
	m.sample(start, 0)

	feed(&m, start, 0, 10*time.Second, tickInterval, 10*megabyte, &bytes)
	_, published := feed(&m, start, 10*time.Second, 25*time.Second, tickInterval, 40*megabyte, &bytes)
	assert.InDelta(t, 40*megabyte, published, megabyte)
}

func TestRestartLeavesAPauseOutOfTheRate(t *testing.T) {
	var m rateMeter
	var bytes int64
	start := time.Unix(0, 0)
	m.sample(start, 0)

	feed(&m, start, 0, 10*time.Second, tickInterval, 10*megabyte, &bytes)
	m.restart(start.Add(40*time.Second), bytes)
	_, published := feed(&m, start, 40*time.Second, 40*time.Second+tickInterval, tickInterval, 10*megabyte, &bytes)
	assert.InDelta(t, 10*megabyte, published, megabyte/100)
}

func TestRateIgnoresAZeroInterval(t *testing.T) {
	var m rateMeter
	var bytes int64
	start := time.Unix(0, 0)
	m.sample(start, 0)

	now, published := feed(&m, start, 0, 5*time.Second, tickInterval, 10*megabyte, &bytes)
	assert.Equal(t, published, m.sample(now, bytes+megabyte))
}
