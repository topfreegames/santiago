# santiago - webhook dispatching service
# https://github.com/topfreegames/santiago
# Licensed under the MIT license:
# http://www.opensource.org/licenses/mit-license
# Copyright © 2016 Top Free Games <backend@tfgco.com>

PACKAGES = $(shell glide novendor)
DIRS = $(shell find . -type f -not -path '*/\.*' | grep '.go' | grep -v "^[.]\/vendor" | xargs -n1 dirname | sort | uniq | grep -v '^.$$')
MYIP = $(shell ifconfig | egrep inet | egrep -v inet6 | egrep -v 127.0.0.1 | awk ' { print $$2 } ')
OS = "$(shell uname | awk '{ print tolower($$0) }')"
LOCAL_REDIS_PORT=57574
REDIS_CONF_PATH=./scripts/redis.conf
TEST_LOCAL_REDIS_PORT=57575
TEST_REDIS_CONF_PATH=./scripts/test_redis.conf

setup: setup-hooks
	@go get -u github.com/ddollar/forego
	@go get -u github.com/onsi/ginkgo/ginkgo
	@go get -u github.com/Masterminds/glide/...
	@glide install

setup-hooks:
	@cd .git/hooks && ln -sf ../../hooks/pre-commit.sh pre-commit

setup-docs:
	@mkdir -p /tmp/.pip/cache
	@pip install -q --log /tmp/pip.log --cache-dir /tmp/.pip/cache --no-cache-dir sphinx recommonmark sphinx_rtd_theme

build:
	@go build $(PACKAGES)
	@mkdir -p bin/
	@go build -o ./bin/snt-worker ./worker/main.go 
	@go build -o ./bin/snt ./main.go 

cross: cross-linux cross-darwin

cross-linux: cross-exec
	@mkdir -p ./bin
	@echo "Building for linux-i386..."
	@env GOOS=linux GOARCH=386 go build -o ./bin/snt-linux-i386 ./main.go
	@env GOOS=linux GOARCH=386 go build -o ./bin/snt-worker-linux-i386 ./worker/main.go
	@echo "Building for linux-x86_64..."
	@env GOOS=linux GOARCH=amd64 go build -o ./bin/snt-linux-x86_64 ./main.go
	@env GOOS=linux GOARCH=amd64 go build -o ./bin/snt-worker-linux-x86_64 ./worker/main.go

cross-darwin: cross-exec
	@mkdir -p ./bin
	@echo "Building for darwin-i386..."
	@env GOOS=darwin GOARCH=386 go build -o ./bin/snt-darwin-i386 ./main.go
	@env GOOS=darwin GOARCH=386 go build -o ./bin/snt-worker-darwin-i386 ./worker/main.go
	@echo "Building for darwin-x86_64..."
	@env GOOS=darwin GOARCH=amd64 go build -o ./bin/snt-darwin-x86_64 ./main.go
	@env GOOS=darwin GOARCH=amd64 go build -o ./bin/snt-worker-darwin-x86_64 ./worker/main.go

cross-exec:
	@chmod +x bin/*

work:
	@go run worker/main.go start -p $(LOCAL_REDIS_PORT) -d

work-prod:
	@./bin/snt-worker-$(OS)-x86_64 start -p $(LOCAL_REDIS_PORT)

run:
	@go run main.go start -p 3333 -d -c ./config/local.yaml

run-prod:
	@./bin/snt-$(OS)-x86_64 start -p 3333 -c ./config/local.yaml

services: redis

services-shutdown: redis-shutdown

services-clear: redis-clear

redis: redis-shutdown
	@if [ -z "$$REDIS_PORT" ]; then \
		redis-server $(REDIS_CONF_PATH) && sleep 1 &&  \
		redis-cli -p $(LOCAL_REDIS_PORT) info > /dev/null && \
		echo "REDIS running locally at localhost:$(LOCAL_REDIS_PORT)."; \
	else \
		echo "REDIS running at $$REDIS_PORT"; \
	fi

redis-shutdown:
	@-redis-cli -p 57574 shutdown

redis-clear:
	@redis-cli -p 57574 FLUSHDB

ci-test:
	@REDIS_PORT=6379 ginkgo --cover $(DIRS)
	@$(MAKE) test-coverage-build

test: test-services
	@ginkgo --cover $(DIRS); \
    case "$$?" in \
	"0") $(MAKE) test-services-shutdown; exit 0;; \
	*) $(MAKE) test-services-shutdown; exit 1;; \
    esac;

test-coverage: test test-coverage-build

test-coverage-build:
	@rm -rf _build
	@mkdir -p _build
	@echo "mode: count" > _build/test-coverage-all.out
	@bash -c 'for f in $$(find . -name "*.coverprofile"); do tail -n +2 $$f >> _build/test-coverage-all.out; done'

test-coverage-html: test-coverage
	@go tool cover -html=_build/test-coverage-all.out

test-services: test-redis

test-services-shutdown: test-redis-shutdown

test-redis: test-redis-shutdown test-redis-clear
	@if [ -z "$$REDIS_PORT" ]; then \
		redis-server $(TEST_REDIS_CONF_PATH) && sleep 1 &&  \
		redis-cli -p $(TEST_LOCAL_REDIS_PORT) info > /dev/null && \
		echo "REDIS running locally at localhost:$(TEST_LOCAL_REDIS_PORT)."; \
	else \
		echo "REDIS running at $$REDIS_PORT"; \
	fi

test-redis-shutdown:
	@-redis-cli -p 57575 shutdown

test-redis-clear:
	@rm -rf "/tmp/redis_test_santiago*"

docker-build:
	@docker build -t santiago .

docker-run:
	@docker run -i -t --rm -e SNT_SERVICES_NSQ_HOST=$(MYIP) -e SNT_SERVICES_NSQ_PORT=6669 -e SNT_SERVICES_NSQLOOKUP_HOST=$(MYIP) -e SNT_SERVICES_NSQLOOKUP_PORT=6667 -p 8080:8080 santiago

docker-dev-build:
	@docker build -t santiago-dev -f ./DevDockerfile .

docker-dev-run:
	@docker run -i -t --rm -p 8080:8080 santiago-dev

rtfd:
	@rm -rf docs/_build
	@sphinx-build -b html -d ./docs/_build/doctrees ./docs/ docs/_build/html
	@open docs/_build/html/index.html
