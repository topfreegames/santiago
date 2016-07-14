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

#RUN curl https://s3.amazonaws.com/bitly-downloads/nsq/nsq-0.3.8.linux-amd64.go1.6.2.tar.gz | tar xz && cp nsq*/bin/* /usr/bin

ADD bin/snt-linux-x86_64 /go/bin/snt
ADD bin/snt-worker-linux-x86_64 /go/bin/snt-worker
RUN chmod +x /go/bin/snt*
RUN mkdir -p /home/santiago/
RUN go get -u github.com/ddollar/forego

ADD ./docker/default.yaml /home/santiago/default.yaml
ADD ./docker/Procfile /home/santiago/Procfile

ENV SNT_SERVICES_NSQ_HOST localhost
ENV SNT_SERVICES_NSQ_PORT 6669
ENV SNT_SERVICES_NSQLOOKUP_HOST localhost
ENV SNT_SERVICES_NSQLOOKUP_PORT 6667

ENTRYPOINT /go/bin/forego start -f /home/santiago/Procfile
