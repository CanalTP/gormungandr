package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"database/sql"

	"github.com/CanalTP/gormungandr"
	"github.com/CanalTP/gormungandr/auth"
	"github.com/CanalTP/gormungandr/internal/schedules"
	"github.com/CanalTP/gormungandr/internal/utils"
	"github.com/CanalTP/gormungandr/kraken"
	cache "github.com/patrickmn/go-cache"
	"github.com/rafaeljesus/rabbus"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/contrib/ginrus"
	"github.com/gin-gonic/gin"
	newrelic "github.com/newrelic/go-agent"
	"github.com/newrelic/go-agent/_integrations/nrgin/v1"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"
	"github.com/spf13/pflag"
)

func setupRouter(config schedules.Config) *gin.Engine {
	r := gin.New()
	r.Use(ginrus.Ginrus(logrus.StandardLogger(), time.RFC3339, false))
	r.Use(gormungandr.InstrumentGin())
	// Recovery middleware recovers from any panics and writes a 500 if there was one.
	r.Use(gormungandr.Recovery())

	r.Use(cors.New(cors.Config{
		AllowAllOrigins:  true,
		AllowHeaders:     []string{"Access-Control-Request-Headers", "Authorization"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	if len(config.NewRelicLicense) > 0 {
		nrConfig := newrelic.NewConfig(config.NewRelicAppName, config.NewRelicLicense)
		app, err := newrelic.NewApplication(nrConfig)
		if err != nil {
			logrus.Fatalf("Impossible to initialize newrelic: %+v", err)
		}
		r.Use(nrgin.Middleware(app))
	}

	r.GET("/metrics", gin.WrapH(promhttp.Handler()))

	r.GET("/", Index)
	r.GET("/status", Status)

	return r
}

func Index(c *gin.Context) {
	//this is temporary, but our LB except this to work...
	c.JSON(200, gin.H{
		"versions": "",
	})
}

func Status(c *gin.Context) {
	c.JSON(200, gin.H{
		"status":  "ok",
		"version": gormungandr.Version,
		"runtime": runtime.Version(),
	})
}

func initLog(jsonLog bool, logLevel string) {
	if jsonLog {
		// Log as JSON instead of the default ASCII formatter.
		logrus.SetFormatter(&logrus.JSONFormatter{})
	}
	logrus.SetOutput(os.Stdout)
	level, err := logrus.ParseLevel(logLevel)
	if err != nil {
		logrus.Fatal(err)
	}
	logrus.SetLevel(level)
}

func main() {
	showVersion := pflag.Bool("version", false, "show version")
	pflag.Parse()
	if *showVersion {
		fmt.Printf("gormungandr %s built with %s", gormungandr.Version, runtime.Version())
		os.Exit(0)
	}

	logger := logrus.WithFields(logrus.Fields{
		"version": gormungandr.Version,
		"runtime": runtime.Version(),
	})
	config, err := schedules.GetConfig()
	if err != nil {
		logger.Fatalf("failure to load configuration: %+v", err)
	}
	initLog(config.JSONLog, config.LogLevel)
	logger = logger.WithFields(logrus.Fields{
		"config": config,
	})
	logger.Info("starting schedules")

	authOption := schedules.SkipAuth()
	var statPublisher *auth.StatPublisher

	if !config.SkipAuth {
		//disable database if authentication isn't used
		var (
			db        *sql.DB
			authCache *cache.Cache
		)
		db, err = sql.Open("postgresInstrumented", config.ConnectionString)
		db.SetMaxOpenConns(config.MaxPostgresqlConnection)
		if err != nil {
			logger.Fatal("connection to postgres failed: ", err)
		}
		err = db.Ping()
		if err != nil {
			logger.Fatal("connection to postgres failed: ", err)
		}
		if config.AuthCacheTimeout.Seconds() > 0 {
			logger.Info("Activate authentication cache")
			authCache = cache.New(config.AuthCacheTimeout, config.AuthCacheTimeout*2)
		}

		authOption = schedules.Auth(auth.AuthenticationMiddleware(db, authCache))
	}

	if len(config.PprofListen) != 0 {
		go func() {
			logrus.Infof("pprof listening on %s", config.PprofListen)
			logger.Error(http.ListenAndServe(config.PprofListen, nil))
		}()
	}

	if !config.SkipStats {
		var rmq *rabbus.Rabbus
		rmq, err = rabbus.New(
			config.RabbitmqDsn,
			rabbus.Durable(true),
			rabbus.Attempts(3),
			rabbus.Sleep(time.Second*2),
		)
		if err != nil {
			logrus.Fatal("failure while connecting to rabbitmq ", err)
		}
		defer func(rmq *rabbus.Rabbus) {
			if err = rmq.Close(); err != nil {
				logrus.Fatal("failure while closing rabbitmq connection ", err)
			}
		}(rmq)
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		go func() {
			if err = rmq.Run(ctx); err != nil && err != context.Canceled {
				logrus.Errorf("rabbus.run ended with error: %+v", err)
			}
		}()
		statPublisher = auth.NewStatPublisher(rmq, config.StatsExchange, 2*time.Second)
	}

	router := setupRouter(config)
	if len(config.KrakenFilesUriStr) > 0 {
		coverages, err := utils.GetFileWithFS(config.KrakenFilesUri)
		if err != nil {
			logger.Fatalf("No coverages: %+v", err)
		}
		krakens := make(map[string]kraken.Kraken)
		for _, coverage := range coverages {
			kraken := kraken.NewKrakenZMQ(coverage.Key, coverage.ZmqSocket, config.Timeout)
			krakens[coverage.Key] = kraken
		}
		schedules.SetupApiMultiCoverage(router, krakens, statPublisher, authOption)
	} else if len(config.Kraken) > 0 {
		kraken := kraken.NewKrakenZMQ("default", config.Kraken, config.Timeout)
		schedules.SetupApi(router, kraken, statPublisher, authOption)
	} else {
		logger.Fatalf("No coverage defined")
		os.Exit(1)
	}

	srv := &http.Server{
		Addr:    config.Listen,
		Handler: router,
	}
	go func() {
		// service connections
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatalf("listen: %s", err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server with
	// a timeout of 5 seconds.
	quit := make(chan os.Signal, 5)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	logrus.Info("Shutdown Server ...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		logrus.Fatal("Server Shutdown:", err)
	}
	logrus.Info("Server exiting")

}
