// santiago - webhook dispatching service
// https://github.com/topfreegames/santiago
// Licensed under the MIT license:
// http://www.opensource.org/licenses/mit-license
// Copyright © 2016 Top Free Games <backend@tfgco.com>

package api

import (
	"encoding/json"
	"fmt"
	"runtime/debug"
	"time"

	"github.com/getsentry/raven-go"
	"github.com/heynemann/myth/metadata"
	"github.com/labstack/echo"
	"github.com/uber-go/zap"
)

//NewVersionMiddleware with API version
func NewVersionMiddleware() *VersionMiddleware {
	return &VersionMiddleware{
		Version: metadata.VERSION,
	}
}

//VersionMiddleware inserts the current version in all requests
type VersionMiddleware struct {
	Version string
}

// Serve serves the middleware
func (v *VersionMiddleware) Serve(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		c.Response().Header().Set(echo.HeaderServer, fmt.Sprintf("Khan/v%s", v.Version))
		return next(c)
	}
}

//NewSentryMiddleware returns a new sentry middleware
func NewSentryMiddleware(app *App) *SentryMiddleware {
	return &SentryMiddleware{
		App: app,
	}
}

//SentryMiddleware is responsible for sending all exceptions to sentry
type SentryMiddleware struct {
	App *App
}

// Serve serves the middleware
func (s *SentryMiddleware) Serve(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		err := next(c)
		if err != nil {
			tags := map[string]string{
				"source": "app",
				"type":   "Internal server error",
				"url":    c.Request().URI(),
				"status": fmt.Sprintf("%d", c.Response().Status()),
			}
			raven.SetHttpContext(newHTTPFromCtx(c))
			raven.CaptureError(err, tags)
		}
		return err
	}
}

func getHTTPParams(ctx echo.Context) (string, map[string]string, string) {
	qs := ""
	if len(ctx.QueryParams()) > 0 {
		qsBytes, _ := json.Marshal(ctx.QueryParams())
		qs = string(qsBytes)
	}

	headers := map[string]string{}
	for _, headerKey := range ctx.Response().Header().Keys() {
		headers[string(headerKey)] = string(ctx.Response().Header().Get(headerKey))
	}

	cookies := string(ctx.Response().Header().Get("Cookie"))
	return qs, headers, cookies
}

func newHTTPFromCtx(ctx echo.Context) *raven.Http {
	qs, headers, cookies := getHTTPParams(ctx)

	h := &raven.Http{
		Method:  string(ctx.Request().Method()),
		Cookies: cookies,
		Query:   qs,
		URL:     ctx.Request().URI(),
		Headers: headers,
	}
	return h
}

//NewRecoveryMiddleware returns a configured middleware
func NewRecoveryMiddleware(onError func(error, []byte)) *RecoveryMiddleware {
	return &RecoveryMiddleware{
		OnError: onError,
	}
}

//RecoveryMiddleware recovers from errors
type RecoveryMiddleware struct {
	OnError func(error, []byte)
}

//Serve executes on error handler when errors happen
func (r *RecoveryMiddleware) Serve(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		defer func() {
			if err := recover(); err != nil {
				eError, ok := err.(error)
				if !ok {
					eError = fmt.Errorf(fmt.Sprintf("%v", err))
				}
				if r.OnError != nil {
					r.OnError(eError, debug.Stack())
				}
				c.Error(eError)
			}
		}()
		return next(c)
	}
}

// NewLoggerMiddleware returns the logger middleware
func NewLoggerMiddleware(theLogger zap.Logger) *LoggerMiddleware {
	l := &LoggerMiddleware{Logger: theLogger}
	return l
}

//LoggerMiddleware is responsible for logging to Zap all requests
type LoggerMiddleware struct {
	Logger zap.Logger
}

// Serve serves the middleware
func (l *LoggerMiddleware) Serve(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		log := l.Logger.With(
			zap.String("source", "request"),
		)

		//all except latency to string
		var ip, method, path string
		var status int
		var latency time.Duration
		var startTime, endTime time.Time

		path = c.Path()
		method = c.Request().Method()

		startTime = time.Now()

		err := next(c)

		//no time.Since in order to format it well after
		endTime = time.Now()
		latency = endTime.Sub(startTime)

		status = c.Response().Status()
		ip = c.Request().RemoteAddress()

		route := c.Get("route")
		if route == nil {
			log.Debug("Route does not have route set in ctx")
			return err
		}

		reqLog := log.With(
			zap.String("route", route.(string)),
			zap.Time("endTime", endTime),
			zap.Int("statusCode", status),
			zap.Duration("latency", latency),
			zap.String("ip", ip),
			zap.String("method", method),
			zap.String("path", path),
		)

		//request failed
		if status > 399 && status < 500 {
			reqLog.Warn("Request failed.")
			return err
		}

		//request is ok, but server failed
		if status > 499 {
			reqLog.Error("Response failed.")
			return err
		}

		//Everything went ok
		if cm := reqLog.Check(zap.InfoLevel, "Request successful."); cm.OK() {
			cm.Write()
		}
		return err
	}
}
