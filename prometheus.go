package gormungandr

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"time"

	"github.com/gchaincl/sqlhooks"
	"github.com/gin-gonic/gin"
	"github.com/lib/pq"
	"github.com/prometheus/client_golang/prometheus"
)

type contextKey string

var (
	sqlBeginKey contextKey = "begin"

	httpDurations = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: "gormungandr",
		Subsystem: "http",
		Name:      "durations_seconds",
		Help:      "http request latency distributions.",
		Buckets:   prometheus.ExponentialBuckets(0.001, 1.5, 25),
	},
		[]string{"handler", "code", "coverage"},
	)

	httpInFlight = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: "gormungandr",
		Subsystem: "http",
		Name:      "in_flight",
		Help:      "current number of http request being served",
	},
	)

	sqlDurations = prometheus.NewHistogram(prometheus.HistogramOpts{
		Namespace: "gormungandr",
		Subsystem: "sql",
		Name:      "durations_seconds",
		Help:      "sql request latency distributions.",
		Buckets:   prometheus.ExponentialBuckets(0.001, 1.5, 15),
	},
	)
)

func init() {
	prometheus.MustRegister(httpDurations)
	prometheus.MustRegister(httpInFlight)
	prometheus.MustRegister(sqlDurations)
	sql.Register("postgresInstrumented", sqlhooks.Wrap(&pq.Driver{}, &instrumentHook{}))
}

func InstrumentGin() gin.HandlerFunc {
	return func(c *gin.Context) {
		begin := time.Now()
		httpInFlight.Inc()
		c.Next()
		httpInFlight.Dec()
		//logrus.Warning(c.Request.RequestURI)
		observer := httpDurations.With(prometheus.Labels{
			"handler":  c.HandlerName(),
			"code":     strconv.Itoa(c.Writer.Status()),
			"coverage": fmt.Sprintf("%s", c.Keys["coverage"])})
		observer.Observe(time.Since(begin).Seconds())
	}
}

// Hooks satisfies the sqlhook.Hooks interface
type instrumentHook struct{}

// register the timestamp in the context
func (h *instrumentHook) Before(ctx context.Context, query string, args ...interface{}) (context.Context, error) {
	return context.WithValue(ctx, sqlBeginKey, time.Now()), nil
}

// After hook will get the timestamp registered on the Before hook and feed prometheus
func (h *instrumentHook) After(ctx context.Context, query string, args ...interface{}) (context.Context, error) {
	begin := ctx.Value(sqlBeginKey).(time.Time)
	sqlDurations.Observe(time.Since(begin).Seconds())
	return ctx, nil
}
