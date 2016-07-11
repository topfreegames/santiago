PACKAGES = $(shell glide novendor)
DIRS = $(shell find ./ -type f -not -path '*/\.*' | grep '.go' | grep -v "^[.]\/vendor" | xargs -n1 dirname | sort | uniq | grep -v '^.$$')

setup-hooks:
	@cd .git/hooks && ln -sf ../../hooks/pre-commit.sh pre-commit

setup: setup-hooks
	@type nsqlookupd >/dev/null 2>&1 || { echo >&2 "Please ensure NSQ is installed before continuing.\nFor more information, refer to http://nsq.io/deployment/installing.html.\n\nSetup aborted!\n"; exit 1; }
	@go get -u github.com/onsi/ginkgo/ginkgo
	@go get -u github.com/Masterminds/glide/...
	@glide install

build:
	@go build $(PACKAGES)

test:
	@ginkgo --cover $(DIRS)

test-coverage: test
	@rm -rf _build
	@mkdir -p _build
	@echo "mode: count" > _build/test-coverage-all.out
	@bash -c 'for f in $$(find . -name "*.coverprofile"); do tail -n +2 $$f >> _build/test-coverage-all.out; done'

test-coverage-html: test-coverage
	@go tool cover -html=_build/test-coverage-all.out
