# santiago - webhook dispatching service
# https://github.com/topfreegames/santiago
# Licensed under the MIT license:
# http://www.opensource.org/licenses/mit-license
# Copyright © 2016 Top Free Games <backend@tfgco.com>

PACKAGES = $(shell glide novendor)
DIRS = $(shell find . -type f -not -path '*/\.*' | grep '.go' | grep -v "^[.]\/vendor" | xargs -n1 dirname | sort | uniq | grep -v '^.$$')

setup-hooks:
	@cd .git/hooks && ln -sf ../../hooks/pre-commit.sh pre-commit

setup: setup-hooks
	@type nsqlookupd >/dev/null 2>&1 || { echo >&2 "Please ensure NSQ is installed before continuing.\nFor more information, refer to http://nsq.io/deployment/installing.html.\n\nSetup aborted!\n"; exit 1; }
	@go get -u github.com/ddollar/forego
	@go get -u github.com/onsi/ginkgo/ginkgo
	@go get -u github.com/Masterminds/glide/...
	@glide install

build:
	@go build $(PACKAGES)
	@mkdir -p bin/
	@go build -o ./bin/snt-worker ./worker/main.go 
	@go build -o ./bin/snt ./main.go 

cross:
	@mkdir -p ./bin
	@echo "Building for linux-i386..."
	@env GOOS=linux GOARCH=386 go build -o ./bin/snt-linux-i386 ./main.go
	@env GOOS=linux GOARCH=386 go build -o ./bin/snt-worker-linux-i386 ./worker/main.go
	@echo "Building for linux-x86_64..."
	@env GOOS=linux GOARCH=amd64 go build -o ./bin/snt-linux-x86_64 ./main.go
	@env GOOS=linux GOARCH=amd64 go build -o ./bin/snt-worker-linux-x86_64 ./worker/main.go
	@echo "Building for darwin-i386..."
	@env GOOS=darwin GOARCH=386 go build -o ./bin/snt-darwin-i386 ./main.go
	@env GOOS=darwin GOARCH=386 go build -o ./bin/snt-worker-darwin-i386 ./worker/main.go
	@echo "Building for darwin-x86_64..."
	@env GOOS=darwin GOARCH=amd64 go build -o ./bin/snt-darwin-x86_64 ./main.go
	@env GOOS=darwin GOARCH=amd64 go build -o ./bin/snt-worker-darwin-x86_64 ./worker/main.go
	@chmod +x bin/*

work:
	@go run worker/main.go start -p 6667 -i 15 -d -m 3

work-prod:
	@go run worker/main.go start -p 6667

run:
	@go run main.go start -c ./config/local.yaml

services: nsq

services-shutdown: nsq-shutdown

services-clear: nsq-clear

nsq: nsq-shutdown
	@rm -rf /tmp/santiago-nsq.log
	@mkdir -p /tmp/nsqd/1
	@mkdir -p /tmp/nsqd/2
	@mkdir -p /tmp/nsqd/3
	@forego start -f ./scripts/NSQProcfile 2>&1 > /tmp/santiago-nsq.log &

nsq-shutdown:
	@-ps aux | egrep forego | egrep -v grep | awk ' { print $$2 } ' | xargs kill -2

nsq-clear:
	@rm -rf /tmp/nsqd

test: test-services
	@ginkgo --cover $(DIRS); \
    case "$$?" in \
	"0") $(MAKE) test-services-shutdown; exit 0;; \
	*) $(MAKE) test-services-shutdown; exit 1;; \
    esac;

test-coverage: test
	@rm -rf _build
	@mkdir -p _build
	@echo "mode: count" > _build/test-coverage-all.out
	@bash -c 'for f in $$(find . -name "*.coverprofile"); do tail -n +2 $$f >> _build/test-coverage-all.out; done'

test-coverage-html: test-coverage
	@go tool cover -html=_build/test-coverage-all.out

test-services: test-nsq

test-services-log: test-nsq-log

test-services-shutdown: test-nsq-shutdown

test-nsq: test-nsq-shutdown test-nsq-clear
	@rm -rf /tmp/santiago-nsq-test.log
	@mkdir -p /tmp/nsqd-test/1
	@forego start -f ./scripts/TestNSQProcfile 2>&1 > /tmp/santiago-nsq-test.log &

test-nsq-shutdown:
	@-ps aux | egrep forego | egrep -v egrep | awk ' { print $$2 } ' | xargs kill -hup

test-nsq-clear:
	@rm -rf /tmp/nsqd-test

test-nsq-log:
	@echo "-------------------------------"
	@echo "NSQ Log:"
	@cat /tmp/santiago-nsq-test.log

docker-build:
	@docker build -t santiago .
