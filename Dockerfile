# santiago - webhook dispatching service
# https://github.com/topfreegames/santiago
# Licensed under the MIT license:
# http://www.opensource.org/licenses/mit-license
# Copyright © 2016 Top Free Games <backend@tfgco.com>

FROM golang:1.6.2-alpine

MAINTAINER TFG Co <backend@tfgco.com>

EXPOSE 8080

RUN apk update
RUN apk add git bash

# http://stackoverflow.com/questions/34729748/installed-go-binary-not-found-in-path-on-alpine-linux-docker
RUN mkdir /lib64 && ln -s /lib/libc.musl-x86_64.so.1 /lib64/ld-linux-x86-64.so.2

ADD bin/snt-linux-x86_64 /go/bin/snt
RUN chmod +x /go/bin/snt*
RUN mkdir -p /home/santiago/

ADD ./docker/default.yaml /home/santiago/default.yaml

ENV SNT_API_REDIS_HOST localhost
ENV SNT_API_REDIS_PORT 6379
ENV SNT_API_REDIS_PASSWORD ""
ENV SNT_API_REDIS_DB 0
ENV SNT_API_SENTRY_URL ""
ENV SNT_USE_FAST_HTTP "--fast"

ENTRYPOINT /go/bin/snt start --host 0.0.0.0 $SNT_USE_FAST_HTTP --port 8080 --config /home/santiago/default.yaml
