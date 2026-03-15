// Package secretary tests for scheduler timezone configuration
package secretary

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSchedulerConfig_Timezone(t *testing.T) {
	cfg := SchedulerConfig{
		Timezone: "America/New_York",
	}

	assert.Equal(t, "America/New_York", cfg.Timezone)
}

func TestScheduler_timezoneAnd_location_fields(t *testing.T) {
	scheduler := &Scheduler{}

	scheduler.timezone = "America/Los_Angeles"
	assert.Equal(t, "America/Los_Angeles", scheduler.timezone)

	loc, err := time.LoadLocation("America/Los_Angeles")
	require.NoError(t, err)
	scheduler.location = loc
	assert.NotNil(t, scheduler.location)
	assert.Equal(t, "America/Los_Angeles", scheduler.location.String())
}

func TestScheduler_TimezoneUTC(t *testing.T) {
	scheduler := &Scheduler{}
	scheduler.timezone = "UTC"

	loc, err := time.LoadLocation("UTC")
	require.NoError(t, err)
	scheduler.location = loc

	assert.Equal(t, "UTC", scheduler.timezone)
	assert.Equal(t, "UTC", scheduler.location.String())
}

func TestScheduler_TimezoneMultipleZones(t *testing.T) {
	zones := []string{
		"UTC",
		"America/New_York",
		"America/Los_Angeles",
		"Europe/London",
		"Asia/Tokyo",
		"Australia/Sydney",
	}

	for _, zone := range zones {
		t.Run(zone, func(t *testing.T) {
			loc, err := time.LoadLocation(zone)
			require.NoError(t, err, "Zone %s should be loadable", zone)
			assert.Equal(t, zone, loc.String())

			now := time.Now().UTC()
			local := now.In(loc)
			assert.NotNil(t, local)
		})
	}
}

func TestScheduler_TimezoneFieldExists(t *testing.T) {
	scheduler := &Scheduler{}

	scheduler.timezone = "America/Chicago"
	scheduler.location = time.UTC

	assert.Equal(t, "America/Chicago", scheduler.timezone)
	assert.NotNil(t, scheduler.location)
}

func TestSchedulerConfig_TimezoneField(t *testing.T) {
	cfg := SchedulerConfig{
		Store:        nil,
		Orchestrator: nil,
		EventEmitter: nil,
		TickInterval: time.Minute,
		Logger:       nil,
		Timezone:     "Europe/Paris",
	}

	assert.Equal(t, "Europe/Paris", cfg.Timezone)
}
