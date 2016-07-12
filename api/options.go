// santiago - webhook dispatching service
// https://github.com/topfreegames/santiago
// Licensed under the MIT license:
// http://www.opensource.org/licenses/mit-license
// Copyright © 2016 Top Free Games <backend@tfgco.com>

package api

//Options specify configuration to start api with
type Options struct {
	Host       string
	Port       int
	Debug      bool
	ConfigFile string
}

//DefaultOptions returns local development options for api
func DefaultOptions() *Options {
	return NewOptions(
		"0.0.0.0",
		3000,
		true,
		"../config/default.yaml",
	)
}

//NewOptions returns new options to create API
func NewOptions(host string, port int, debug bool, configFile string) *Options {
	return &Options{
		Host:       host,
		Port:       port,
		Debug:      debug,
		ConfigFile: configFile,
	}
}
