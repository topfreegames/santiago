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

	"github.com/iris-contrib/middleware/recovery"
	"github.com/kataras/iris"
	"github.com/kataras/iris/config"
	"github.com/rcrowley/go-metrics"
	"github.com/spf13/viper"
	"github.com/uber-go/zap"
)

//App is responsible for Santiago's API
type App struct {
	Config        *viper.Viper
	Logger        zap.Logger
	ServerOptions *Options
	WebApp        *iris.Framework
	Client        *redis.Client
	Queue         string
	Errors        metrics.EWMA
}

//New opens a new channel connection
func New(options *Options, logger zap.Logger) (*App, error) {
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
	l.Debug("Initializing app...")

	a.setDefaultConfigurationOptions()

	err := a.loadConfiguration()
	if err != nil {
		return err
	}

	a.connectToRedis()
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

	l.Debug("Connecting to Redis...")
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
	l.Info("Connected to Redis successfully.", zap.Duration("connection", time.Now().Sub(start)))

	a.Client = client
	return nil
}

func (a *App) initializeWebApp() {
	debug := a.ServerOptions.Debug

	l := a.Logger.With(
		zap.String("operation", "loadConfiguration"),
		zap.Bool("debug", debug),
	)

	c := config.Iris{
		DisableBanner: true,
	}

	a.WebApp = iris.New(c)

	a.WebApp.Use(NewLoggerMiddleware(a.Logger))
	a.WebApp.Use(&RecoveryMiddleware{OnError: a.onErrorHandler})
	a.WebApp.Use(&VersionMiddleware{App: a})

	a.WebApp.Use(NewLoggerMiddleware(a.Logger))
	a.WebApp.Use(recovery.New(os.Stderr))

	a.WebApp.Get("/healthcheck", HealthCheckHandler(a))
	a.WebApp.Get("/status", StatusHandler(a))
	a.WebApp.Post("/hooks", AddHookHandler(a))

	l.Info("Web App configured successfully")
}

//GetMessageCount returns the message count for the queue
func (a *App) GetMessageCount() (int, error) {
	queue := a.Queue

	l := a.Logger.With(
		zap.String("operation", "GetMessageCount"),
		zap.Object("queue", queue),
	)

	l.Debug("Getting message count...")

	total, err := a.Client.LLen(queue).Result()
	if err != nil {
		return 0, err
	}

	messageCount := int(total)
	l.Debug("Message count retrieved successfully.", zap.Int("messageCount", messageCount))

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

	l.Debug("Publishing hook...")
	_, err := a.Client.RPush(queue, dataJSON).Result()
	if err != nil {
		l.Error("Publishing hook failed.", zap.Error(err))
		return err
	}
	l.Info("Hook published successfully.", zap.Duration("PublishDuration", time.Now().Sub(start)))

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
}

//Start the application
func (a *App) Start() {
	l := a.Logger.With(
		zap.String("operation", "Start"),
	)

	bind := fmt.Sprintf("%s:%d", a.ServerOptions.Host, a.ServerOptions.Port)
	l.Info("Listening for requests.", zap.String("bind", bind))
	a.WebApp.Listen(bind)
}
