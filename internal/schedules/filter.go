package schedules

import (
	"net/http"

	"github.com/CanalTP/gormungandr"
	"github.com/CanalTP/gormungandr/kraken"
	"github.com/gin-gonic/gin"
)

func NoRouteHandler(kraken kraken.Kraken, coverageName string, publisher Publisher) gin.HandlerFunc {
	fn := func(c *gin.Context) {
		request := gormungandr.NewRequest()
		c.Header("navitia-request-id", request.ID.String())
		filter, err := gormungandr.ParsePath(c.Param("filter"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err})
			return
		}
		if user, ok := gormungandr.GetUser(c); ok {
			request.User = user
		}
		request.Coverage = c.Param("coverage")
		if len(coverageName) > 0 {
			c.Keys["coverage"] = coverageName
		}

		if filter.API == "route_schedules" {
			request := NewRouteScheduleRequest(request)
			if err := c.ShouldBindQuery(&request); err != nil {
				request.Logger().Debugf("%+v\n", err)
				c.JSON(http.StatusBadRequest, gin.H{"error": err})
				return
			}
			request.ForbiddenUris = append(request.ForbiddenUris, c.QueryArray("forbidden_uris[]")...)
			request.Filters = append(request.Filters, filter.Filters...)
			if len(request.Filters) < 1 {
				c.JSON(http.StatusNotFound, gin.H{"message": "at least one filter is required"})
				return
			}

			RouteSchedule(c, kraken, &request, publisher)
		} else {
			c.JSON(http.StatusNotFound, gin.H{"error": "API not found"})
			return
		}
	}
	return gin.HandlerFunc(fn)
}
