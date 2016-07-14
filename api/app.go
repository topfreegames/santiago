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

	"github.com/iris-contrib/middleware/recovery"
	"github.com/kataras/fasthttp"
	"github.com/kataras/iris"
	"github.com/kataras/iris/config"
	"github.com/spf13/viper"
	"github.com/uber-go/zap"
)

//App is responsible for Santiago's API
type App struct {
	Config        *viper.Viper
	Logger        zap.Logger
	ServerOptions *Options
	WebApp        *iris.Framework
	Queue         string
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
	}
	a.Queue = "webhooks"

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

	a.initializeWebApp()

	l.Info(
		"App initialized successfully.",
		zap.Duration("appInitialization", time.Now().Sub(start)),
	)

	return nil
}

func (a *App) setDefaultConfigurationOptions() {
	a.Config.SetDefault("api.workingText", "WORKING")
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

//DoRequest to some webhook endpoint
func (a *App) DoRequest(method, url, payload string) (int, string, error) {
	l := a.Logger.With(
		zap.String("operation", "DoRequest"),
		zap.String("method", method),
		zap.String("url", url),
		zap.String("payload", payload),
	)

	client := fasthttp.Client{
		Name: "santiago",
	}

	req := fasthttp.AcquireRequest()
	req.Header.SetMethod(method)
	req.SetRequestURI(url)
	req.AppendBody([]byte(payload))
	resp := fasthttp.AcquireResponse()

	timeout := time.Duration(5) * time.Second

	start := time.Now()
	l.Debug("Sending to NSQ...")
	err := client.DoTimeout(req, resp, timeout)
	if err != nil {
		l.Error("Request to NSQ failed.", zap.Error(err))
		return 0, "", err
	}

	l.Info("Sending to NSQ succeeded.", zap.Duration("RequestDuration", time.Now().Sub(start)))
	return resp.StatusCode(), string(resp.Body()), nil
}

//PublishHook sends a hook to NSQ
func (a *App) PublishHook(method, url string, payload string) error {
	host := a.Config.GetString("services.NSQ.host")
	port := a.Config.GetInt("services.NSQ.port")
	nsqURL := fmt.Sprintf("http://%s:%d/put?topic=%s", host, port, a.Queue)

	l := a.Logger.With(
		zap.String("operation", "PublishHook"),
		zap.String("url", url),
		zap.Object("payload", payload),
		zap.Object("nsqURL", nsqURL),
	)

	data := map[string]interface{}{
		"method":  method,
		"url":     url,
		"payload": payload,
	}
	dataJSON, _ := json.Marshal(data)

	start := time.Now()
	l.Debug("Publishing hook...")
	status, _, err := a.DoRequest("POST", nsqURL, string(dataJSON))
	if err != nil {
		l.Error("Publishing hook failed.", zap.Error(err))
		return err
	}
	if status > 399 {
		err := fmt.Errorf("Could not add hook to queue at %s (status: %d)", nsqURL, status)
		l.Error("Publishing hook failed.", zap.Error(err))
		return err
	}
	l.Info("Hook published successfully.", zap.Duration("PublishDuration", time.Now().Sub(start)))

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
	a.WebApp.Use(recovery.New(os.Stderr))

	a.WebApp.Get("/healthcheck", HealthCheckHandler(a))
	a.WebApp.Post("/hooks", AddHookHandler(a))

	l.Info("Web App configured successfully")
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
