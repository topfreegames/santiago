Hosting Santiago
================

There are three ways to host Santiago: docker, binaries or from source.

## Docker

Running santiago with docker is rather simple. Our docker container image already comes bundled with both the API and the Worker. All you need to do is load balance all the containers and you're good to go.

Santiago uses Redis to publish hooks to and to listen for incoming hooks. The container also takes parameters to specify this connection:

* `SNT_API_REDIS_HOST` - Redis host to publish hooks to;
* `SNT_API_REDIS_PORT` - Redis port to publish hooks to;
* `SNT_API_REDIS_PASSWORD` - Password of the Redis Server to listen for hooks;
* `SNT_API_REDIS_DB` - DB Number of the Redis Server to listen for hooks;
* `SNT_API_USE_FAST_HTTP` - Whether to use fasthttp for echo engine or not. This env should be either "--fast" or "".
* `SNT_NEWRELIC_KEY` - New Relic account key. If present will enable New Relic.

## Binaries

Whenever we publish a new version of Santiago, we'll always supply binaries for both Linux and Darwing, on i386 and x86_64 architectures. If you'd rather run your own servers instead of containers, just use the binaries that match your platform and architecture.

The API server is the `snt` binary. It takes a configuration yaml file that specifies the connection to Redis and some additional parameters. You can learn more about it at [default.yaml](https://github.com/topfreegames/santiago/blob/master/config/default.yaml).

The workers are started by the `snt-worker` binary. This one takes all the parameters it needs via console options. To learn what options are available, use `snt-worker -h`. To start a new worker, use `snt-worker start`.

## Source

Left as an exercise to the reader.
