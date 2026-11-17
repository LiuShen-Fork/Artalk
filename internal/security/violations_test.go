package security

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestViolationTrackerCountsWithinWindow(t *testing.T) {
	tracker := NewViolationTracker()
	now := time.Date(2026, 11, 17, 12, 0, 0, 0, time.UTC)
	window := time.Hour

	assert.Equal(t, 1, tracker.Add("1.2.3.4", now, window))
	assert.Equal(t, 2, tracker.Add("1.2.3.4", now.Add(time.Minute), window))
	assert.Equal(t, 3, tracker.Add("1.2.3.4", now.Add(2*time.Minute), window))

	assert.Equal(t, 3, tracker.Count("1.2.3.4", window, now.Add(2*time.Minute)))
	assert.Equal(t, 0, tracker.Count("5.6.7.8", window, now))
	assert.Equal(t, 1, tracker.Len())
}

func TestViolationTrackerDropsOldEntries(t *testing.T) {
	tracker := NewViolationTracker()
	now := time.Date(2026, 11, 17, 12, 0, 0, 0, time.UTC)
	window := time.Hour

	tracker.Add("1.2.3.4", now, window)
	tracker.Add("1.2.3.4", now.Add(30*time.Minute), window)

	later := now.Add(90 * time.Minute)
	assert.Equal(t, 1, tracker.Count("1.2.3.4", window, later), "the first entry falls out of the window")
	assert.Equal(t, 2, tracker.Add("1.2.3.4", later, window), "Add counts only entries inside the window")
}

func TestViolationTrackerReset(t *testing.T) {
	tracker := NewViolationTracker()
	now := time.Date(2026, 11, 17, 12, 0, 0, 0, time.UTC)

	tracker.Add("1.2.3.4", now, time.Hour)
	tracker.Reset("1.2.3.4")

	assert.Equal(t, 0, tracker.Count("1.2.3.4", time.Hour, now))
	assert.Equal(t, 0, tracker.Len())
}
