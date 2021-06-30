package gormungandr

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseProtect(t *testing.T) {
	assert.Equal(t, "\"\"", protect(""))
	assert.Equal(t, "\"stop_area:OIF:SA:8768600\"", protect("stop_area:OIF:SA:8768600"))
	assert.Equal(t, "\"stop_area:OIF:SA:8768600\"", protect("stop_area:OIF:SA:8768600"))
	assert.Equal(t, "\"stop_area:OIFé&ù:SA:8768600\"", protect("stop_area:OIFé&ù:SA:8768600"))
}

func TestParsePathNil(t *testing.T) {
	_, err := ParsePath("")
	assert.NotNil(t, err)
}

func TestConvertToFilter(t *testing.T) {
	_, err := convertToFilter("", "")
	assert.NotNil(t, err)

	_, err = convertToFilter("foo", "")
	assert.NotNil(t, err)

	_, err = convertToFilter("coord", "")
	assert.NotNil(t, err)

	r, err := convertToFilter("stop_areas", "foo")
	assert.Nil(t, err)
	assert.Equal(t, "stop_area.uri=\"foo\"", r)

	r, err = convertToFilter("lines", "foo")
	assert.Nil(t, err)
	assert.Equal(t, "line.uri=\"foo\"", r)
}

func TestParsePathRouteSchedules(t *testing.T) {
	f, err := ParsePath("/route_schedules")
	assert.Nil(t, err)
	assert.Equal(t, "route_schedules", f.API)
	assert.Empty(t, f.Filters)

	f, err = ParsePath("/stop_areas/stop_area:OIF:SA:8768600/route_schedules")
	assert.Nil(t, err)
	assert.Equal(t, "route_schedules", f.API)
	assert.Equal(t, 1, len(f.Filters))
	assert.Equal(t, "stop_area.uri=\"stop_area:OIF:SA:8768600\"", f.Filters[0])

	f, err = ParsePath("/stop_areas/stop_area:OIF:SA:8768600/stop_areas/stop_area:OIF:SA:8727100/route_schedules")
	assert.Nil(t, err)
	assert.Equal(t, "route_schedules", f.API)
	assert.Equal(t, 2, len(f.Filters))
	assert.Equal(t, "stop_area.uri=\"stop_area:OIF:SA:8768600\"", f.Filters[0])
	assert.Equal(t, "stop_area.uri=\"stop_area:OIF:SA:8727100\"", f.Filters[1])

	f, err = ParsePath("/lines/line:2/stop_areas/stop_area:OIF:SA:8727100/route_schedules")
	assert.Nil(t, err)
	assert.Equal(t, "route_schedules", f.API)
	assert.Equal(t, 2, len(f.Filters))
	assert.Equal(t, "line.uri=\"line:2\"", f.Filters[0])
	assert.Equal(t, "stop_area.uri=\"stop_area:OIF:SA:8727100\"", f.Filters[1])
}
