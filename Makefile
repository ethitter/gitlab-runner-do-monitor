PROJECT_NAME := "gitlab-runner-do-monitor"
PKG := "git.ethitter.com/debian/$(PROJECT_NAME)"
PKG_LIST := $(shell go list ${PKG}/... | grep -v /vendor/)
GO_FILES := $(shell find . -name '*.go' | grep -v /vendor/ | grep -v _test.go)

.PHONY: all dep build clean test coverage coverhtml lint

all: build

lint:
	@golint -set_exit_status ${PKG_LIST}

test:
	@go get -u github.com/jstemmer/go-junit-report
	@go test -v ${PKG_LIST}
	@go test -v 2>&1 | go-junit-report > report.xml

race: dep
	@go test -v -race ${PKG_LIST}

msan: dep
	@go test -v -msan ${PKG_LIST}

coverage:
	./tools/coverage.sh;

coverhtml:
	./tools/coverage.sh html;

dep:
	@go get -v -d ./...

build: dep
	@go get github.com/mitchellh/gox
	@gox -output="${CI_PROJECT_DIR}/${PROJECT_NAME}/{{.Dir}}_{{.OS}}_{{.Arch}}" -parallel=6

clean:
	@rm -f $(PROJECT_NAME)

help:
	@grep -h -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'
