package schedules

import (
	"encoding/json"
	"flag"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/CanalTP/gonavitia"
	"github.com/CanalTP/gormungandr/internal/checker"
	"github.com/CanalTP/gormungandr/kraken"
	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var departureBoardTest kraken.Kraken
var mainRoutingTest kraken.Kraken

func init() {
	gin.SetMode(gin.TestMode)
}

func TestMain(m *testing.M) {
	flag.Parse() //required to get Short() from testing
	if testing.Short() {
		log.Warn("skipping test Docker in short mode.")
		os.Exit(m.Run())
	}

	mockManager, err := checker.NewMockManager()
	if err != nil {
		log.Fatalf("Could not initialize mocks: %s", err)
	}
	departureBoardTest, err = mockManager.DepartureBoardTest()
	if err != nil {
		log.Fatalf("Could not start departure_board_test: %s", err)
	}

	mainRoutingTest, err = mockManager.MainRoutingTest()
	if err != nil {
		log.Fatalf("Could not start main_routing_test: %s", err)
	}
	//Run tests
	code := m.Run()

	// You can't defer this because os.Exit doesn't care for defer
	mockManager.Close()

	os.Exit(code)
}

func TestRouteSchedules(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test Docker in short mode.")
	}
	t.Parallel()
	assert := assert.New(t)
	require := require.New(t)
	c, engine := gin.CreateTestContext(httptest.NewRecorder())
	SetupApi(engine, departureBoardTest, &NullPublisher{}, SkipAuth())

	c.Request = httptest.NewRequest("GET", "http://api.navitia.io/v1/coverage/foo/routes/line:A:0/route_schedules?from_datetime=20120615T080000", nil)
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, c.Request)
	require.Equal(200, w.Code)

	var response gonavitia.RouteScheduleResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.Nil(err)

	require.Len(response.RouteSchedules, 1)
	require.NotNil(response.Context)
	assert.Equal("UTC", response.Context.Timezone)
	schedules := response.RouteSchedules[0]

	links := make(map[string]string)
	for _, l := range response.Links {
		links[*l.Type] = *l.Href
	}
	//check that the base URL is valid
	require.Contains(links, "first")
	require.Contains(links, "last")
	assert.Contains(links["first"], "http://api.navitia.io/")
	assert.Contains(links["last"], "http://api.navitia.io/")

	checker.IsValidRouteSchedule(t, schedules)

	scheduleLinks := make(map[string]string)
	for _, l := range schedules.Links {
		scheduleLinks[*l.Type] = *l.Id
	}
	assert.Equal("line:A", scheduleLinks["line"])
	assert.Equal("line:A:0", scheduleLinks["route"])
	assert.Equal("base_network", scheduleLinks["network"])

	require.Len(schedules.Table.Headers, 4)

	headsigns := []string{}
	headerByHeadsign := make(map[string]*gonavitia.Header)
	for _, h := range schedules.Table.Headers {
		headsigns = append(headsigns, *h.DisplayInfo.Headsign)
		headerByHeadsign[*h.DisplayInfo.Headsign] = h
	}
	assert.ElementsMatch([]string{"week", "week_bis", "all", "wednesday"}, headsigns)

	headerLinks := make(map[string]string)
	for _, l := range headerByHeadsign["all"].Links {
		headerLinks[*l.Type] = *l.Id
	}
	assert.Equal("vehicle_journey:all", headerLinks["vehicle_journey"])
	assert.Equal("physical_mode:0", headerLinks["physical_mode"])

	//TODO tests on notes when implemented

}

func TestRouteSchedulesHeadsign(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test Docker in short mode.")
	}
	t.Parallel()
	assert := assert.New(t)
	require := require.New(t)
	c, engine := gin.CreateTestContext(httptest.NewRecorder())
	SetupApi(engine, mainRoutingTest, &NullPublisher{}, SkipAuth())

	c.Request = httptest.NewRequest("GET", "/v1/coverage/foo/routes/A:0/route_schedules?from_datetime=20120615T000000", nil)
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, c.Request)
	require.Equal(200, w.Code)

	var response gonavitia.RouteScheduleResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.Nil(err)
	assert.Nil(response.Error)

	require.Len(response.RouteSchedules, 1)
	schedule := response.RouteSchedules[0]
	checker.IsValidRouteSchedule(t, schedule)
	require.Len(schedule.Table.Headers, 1)
	require.NotNil(schedule.Table.Headers[0].DisplayInfo)
	displayInfo := schedule.Table.Headers[0].DisplayInfo
	require.NotNil(displayInfo.Headsign)
	assert.Equal("vjA", *displayInfo.Headsign)
	assert.ElementsMatch([]string{"A00", "vjA"}, displayInfo.Headsigns)
}

func TestRouteSchedulesDisruptions(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test Docker in short mode.")
	}
	t.Parallel()
	assert := assert.New(t)
	require := require.New(t)
	c, engine := gin.CreateTestContext(httptest.NewRecorder())
	SetupApi(engine, mainRoutingTest, &NullPublisher{}, SkipAuth())

	c.Request = httptest.NewRequest("GET", "/v1/coverage/foo/lines/A/route_schedules?from_datetime=20120801T000000&_current_datetime=20120801T050000", nil)
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, c.Request)
	require.Equal(200, w.Code)

	var response gonavitia.RouteScheduleResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.Nil(err)
	assert.Nil(response.Error)
	require.Len(response.RouteSchedules, 2)
	//TODO add more tests when handling disruptions
}

func TestMultiCoverage(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test Docker in short mode.")
	}
	t.Parallel()
	assert := assert.New(t)
	require := require.New(t)
	c, engine := gin.CreateTestContext(httptest.NewRecorder())
	krakens := map[string]kraken.Kraken{"cov1": departureBoardTest, "cov2": mainRoutingTest}
	SetupApiMultiCoverage(engine, krakens, &NullPublisher{}, SkipAuth())

	//time.Sleep(10 * time.Second)

	c.Request = httptest.NewRequest("GET", "http://api.navitia.io/v1/coverage/cov1/routes/line:A:0/route_schedules?from_datetime=20120615T080000", nil)
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, c.Request)
	require.Equal(200, w.Code)

	var response gonavitia.RouteScheduleResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.Nil(err)

	require.Len(response.RouteSchedules, 1)
	require.NotNil(response.Context)

	c.Request = httptest.NewRequest("GET", "/v1/coverage/cov2/routes/A:0/route_schedules?from_datetime=20120615T000000", nil)
	w = httptest.NewRecorder()
	engine.ServeHTTP(w, c.Request)
	require.Equal(200, w.Code)

	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.Nil(err)
	assert.Nil(response.Error)

	require.Len(response.RouteSchedules, 1)
	schedule := response.RouteSchedules[0]
	checker.IsValidRouteSchedule(t, schedule)
	require.Len(schedule.Table.Headers, 1)
	require.NotNil(schedule.Table.Headers[0].DisplayInfo)
	displayInfo := schedule.Table.Headers[0].DisplayInfo
	require.NotNil(displayInfo.Headsign)
	assert.Equal("vjA", *displayInfo.Headsign)
	assert.ElementsMatch([]string{"A00", "vjA"}, displayInfo.Headsigns)
}
