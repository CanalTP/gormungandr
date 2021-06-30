package schedules

import (
	"fmt"

	"github.com/CanalTP/gonavitia"
	"github.com/CanalTP/gormungandr/kraken"
	"github.com/gin-contrib/location"
	"github.com/gin-gonic/gin"
)

type Publisher interface {
	PublishRouteSchedule(request RouteScheduleRequest, response gonavitia.RouteScheduleResponse, c gin.Context) error
}

type NullPublisher struct{}

func (p *NullPublisher) PublishRouteSchedule(request RouteScheduleRequest, response gonavitia.RouteScheduleResponse, c gin.Context) error {
	return nil
}

type AuthOption func(*gin.RouterGroup)

func Auth(authMiddleware gin.HandlerFunc) AuthOption {
	return func(group *gin.RouterGroup) {
		group.Use(authMiddleware)
	}
}
func SkipAuth() AuthOption {
	return func(group *gin.RouterGroup) {}
}

func SetupApi(router *gin.Engine, kraken kraken.Kraken, statPublisher Publisher, auth AuthOption) {
	// middleware must be define before handlers
	router.Use(location.New(location.Config{
		Scheme: "http",
		Host:   "navitia.io",
	}))

	cov := router.Group("/v1/coverage/:coverage")
	auth(cov)
	cov.GET("/*filter", NoRouteHandler(kraken, "", statPublisher))
}

func SetupApiMultiCoverage(router *gin.Engine, krakens map[string]kraken.Kraken, statPublisher Publisher, auth AuthOption) {
	// middleware must be define before handlers
	router.Use(location.New(location.Config{
		Scheme: "http",
		Host:   "navitia.io",
	}))

	for coverageName, kraken := range krakens {
		cov := router.Group(fmt.Sprintf("/v1/coverage/%s", coverageName))
		auth(cov)
		cov.GET("/*filter", NoRouteHandler(kraken, coverageName, statPublisher))
	}
}
