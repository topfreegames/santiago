// santiago - webhook dispatching service
// https://github.com/topfreegames/santiago
// Licensed under the MIT license:
// http://www.opensource.org/licenses/mit-license
// Copyright © 2016 Top Free Games <backend@tfgco.com>

package api

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/redis.v4"

	"github.com/getsentry/raven-go"
	"github.com/labstack/echo"
	"github.com/labstack/echo/engine"
	"github.com/labstack/echo/engine/fasthttp"
	"github.com/labstack/echo/engine/standard"
	"github.com/rcrowley/go-metrics"
	"github.com/spf13/viper"
	"github.com/topfreegames/santiago/log"
	"github.com/uber-go/zap"
)

//App is responsible for Santiago's API
type App struct {
	Fast          bool
	Config        *viper.Viper
	Logger        zap.Logger
	ServerOptions *Options
	Engine        engine.Server
	WebApp        *echo.Echo
	Client        *redis.Client
	Queue         string
	Errors        metrics.EWMA
}

//New opens a new channel connection
func New(options *Options, logger zap.Logger, fast bool) (*App, error) {
	if options == nil {
		options = DefaultOptions()
	}
	l := logger.With(
		zap.String("source", "api"),
		zap.String("host", options.Host),
		zap.Int("port", options.Port),
		zap.Bool("debug", options.Debug),
	)
	a := App{
		Logger:        l,
		ServerOptions: options,
		Config:        viper.New(),
		Fast:          fast,
		Queue:         "webhooks",
	}

	err := a.initialize()
	if err != nil {
		return nil, err
	}

	return &a, nil
}

func (a *App) initialize() error {
	l := a.Logger.With(
		zap.String("operation", "initialize"),
	)
	start := time.Now()
	log.D(l, "Initializing app...")

	a.setDefaultConfigurationOptions()

	err := a.loadConfiguration()
	if err != nil {
		return err
	}

	err = a.connectToRedis()
	if err != nil {
		return err
	}
	a.connectRaven()
	a.initializeWebApp()

	l.Info(
		"App initialized successfully.",
		zap.Duration("appInitialization", time.Now().Sub(start)),
	)

	a.Errors = metrics.NewEWMA15()

	go func(app *App) {
		app.Errors.Tick()
		time.Sleep(5 * time.Second)
	}(a)

	return nil
}

func (a *App) setDefaultConfigurationOptions() {
	a.Config.SetDefault("api.workingText", "WORKING")

	a.Config.SetDefault("api.redis.host", "localhost")
	a.Config.SetDefault("api.redis.port", 57575)
	a.Config.SetDefault("api.redis.password", "")
	a.Config.SetDefault("api.redis.db", 0)

	a.Config.SetDefault("api.sentry.url", "")
}

func (a *App) loadConfiguration() error {
	l := a.Logger.With(
		zap.String("operation", "loadConfiguration"),
		zap.String("configFile", a.ServerOptions.ConfigFile),
	)

	absConfigFile, err := filepath.Abs(a.ServerOptions.ConfigFile)
	if err != nil {
		l.Error("Configuration file not found.", zap.Error(err))
		return err
	}

	l = l.With(
		zap.String("absConfigFile", absConfigFile),
	)

	l.Info("Loading configuration.")

	if _, err := os.Stat(absConfigFile); os.IsNotExist(err) {
		l.Error("Configuration file not found.", zap.Error(err))
		return err
	}

	a.Config.SetConfigFile(a.ServerOptions.ConfigFile)
	a.Config.SetEnvPrefix("snt") // read in environment variables that match
	a.Config.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	a.Config.AutomaticEnv()

	// If a config file is found, read it in.
	if err := a.Config.ReadInConfig(); err != nil {
		l.Error("Configuration could not be loaded.", zap.Error(err))
		return err
	}

	l.Info(
		"Configuration loaded successfully.",
		zap.String("configPath", a.Config.ConfigFileUsed()),
	)
	return nil
}

func (a *App) connectToRedis() error {
	redisHost := a.Config.GetString("api.redis.host")
	redisPort := a.Config.GetInt("api.redis.port")
	redisPass := a.Config.GetString("api.redis.password")
	redisDB := a.Config.GetInt("api.redis.db")

	l := a.Logger.With(
		zap.String("source", "api"),
		zap.String("operation", "connectToRedis"),
		zap.String("redisHost", redisHost),
		zap.Int("redisPort", redisPort),
		zap.Int("redisDB", redisDB),
	)

	log.D(l, "Connecting to Redis...")
	client := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", redisHost, redisPort),
		Password: redisPass,
		DB:       redisDB,
	})

	start := time.Now()
	_, err := client.Ping().Result()
	if err != nil {
		l.Error("Could not connect to redis.", zap.Error(err))
		return err
	}
	log.I(l, "Connected to Redis successfully.", func(cm log.CM) {
		cm.Write(zap.Duration("connection", time.Now().Sub(start)))
	})

	a.Client = client
	return nil
}

func (a *App) connectRaven() {
	raven.SetDSN(a.Config.GetString("api.sentry.url"))
}

func (a *App) initializeWebApp() {
	debug := a.ServerOptions.Debug

	l := a.Logger.With(
		zap.String("operation", "loadConfiguration"),
		zap.Bool("debug", debug),
	)

	a.Engine = standard.New(fmt.Sprintf("%s:%d", a.ServerOptions.Host, a.ServerOptions.Port))
	if a.Fast {
		engine := fasthttp.New(fmt.Sprintf("%s:%d", a.ServerOptions.Host, a.ServerOptions.Port))
		engine.ReadBufferSize = 30000
		a.Engine = engine
	}
	a.WebApp = echo.New()

	a.WebApp.Use(NewLoggerMiddleware(a.Logger).Serve)
	a.WebApp.Use(NewRecoveryMiddleware(a.onErrorHandler).Serve)
	a.WebApp.Use(NewVersionMiddleware().Serve)
	a.WebApp.Use(NewSentryMiddleware(a).Serve)

	a.WebApp.Get("/healthcheck", HealthCheckHandler(a))
	a.WebApp.Get("/status", StatusHandler(a))
	a.WebApp.Post("/hooks", AddHookHandler(a))

	log.I(l, "Web App configured successfully")
}

//GetMessageCount returns the message count for the queue
func (a *App) GetMessageCount() (int, error) {
	queue := a.Queue

	l := a.Logger.With(
		zap.String("operation", "GetMessageCount"),
		zap.Object("queue", queue),
	)

	log.D(l, "Getting message count...")

	total, err := a.Client.LLen(queue).Result()
	if err != nil {
		return 0, err
	}

	messageCount := int(total)
	log.D(l, "Message count retrieved successfully.", func(cm log.CM) {
		cm.Write(zap.Int("messageCount", messageCount))
	})

	return messageCount, nil
}

//PublishHook sends a hook to the queue
func (a *App) PublishHook(method, url string, payload string) error {
	queue := a.Queue

	l := a.Logger.With(
		zap.String("operation", "PublishHook"),
		zap.String("url", url),
		zap.Object("payload", payload),
		zap.Object("queue", queue),
	)

	data := map[string]interface{}{
		"method":   method,
		"url":      url,
		"payload":  payload,
		"attempts": 0,
	}
	dataJSON, _ := json.Marshal(data)

	start := time.Now()

	log.D(l, "Publishing hook...")
	_, err := a.Client.RPush(queue, dataJSON).Result()
	if err != nil {
		l.Error("Publishing hook failed.", zap.Error(err))
		return err
	}
	log.I(l, "Hook published successfully.", func(cm log.CM) {
		cm.Write(zap.Duration("PublishDuration", time.Now().Sub(start)))
	})

	return nil
}

func (a *App) onErrorHandler(err error, stack []byte) {
	a.Errors.Update(1)
	a.Logger.Error(
		"Panic occurred.",
		zap.String("source", "app"),
		zap.String("panicText", err.Error()),
		zap.String("stack", string(stack)),
	)
	tags := map[string]string{
		"source": "app",
		"type":   "panic",
	}
	raven.CaptureError(err, tags)
}

//Start the application
func (a *App) Start() {
	l := a.Logger.With(
		zap.String("operation", "Start"),
	)

	bind := fmt.Sprintf("%s:%d", a.ServerOptions.Host, a.ServerOptions.Port)
	log.I(l, "Listening for requests.", func(cm log.CM) {
		cm.Write(zap.String("bind", bind))
	})
	a.WebApp.Run(a.Engine)
}
