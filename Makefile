.DEFAULT_GOAL  := build
CMD            := wantarr
GOARCH         := $(shell go env GOARCH)
GOOS           := $(shell go env GOOS)
TARGET         := ${GOOS}_${GOARCH}
DIST_PATH      := dist
BUILD_PATH     := ${DIST_PATH}/${CMD}_${TARGET}
DESTDIR        := /usr/local/bin
GO_FILES       := $(shell find . -path ./vendor -prune -or -type f -name '*.go' -print)
GO_PACKAGES    := $(shell go list -mod vendor ./...)
GIT_COMMIT     := $(shell git rev-parse --short HEAD)
# GIT_BRANCH     := $(shell git symbolic-ref --short HEAD)
TIMESTAMP      := $(shell date +%s)
VERSION        ?= 0.0.0-dev

# Deps
.PHONY: check_golangci
check_golangci:
	@command -v golangci-lint >/dev/null || (echo "golangci-lint is required."; exit 1)
.PHONY: check_goreleaser
check_goreleaser:
	@command -v goreleaser >/dev/null || (echo "goreleaser is required."; exit 1)

.PHONY: all ## Run tests, linting and build
all: test lint build

.PHONY: test-all ## Run tests and linting
test-all: test lint

.PHONY: test
test: ## Run tests
	@echo "*** go test ***"
	go test -cover -v -race ${GO_PACKAGES}

.PHONY: lint
lint: check_golangci ## Run linting
	@echo "*** golangci-lint ***"
	golangci-lint run

.PHONY: vendor
vendor: ## Vendor files and tidy go.mod
	go mod vendor
	go mod tidy

.PHONY: vendor_update
vendor_update: ## Update vendor dependencies
	go get -u ./...
	${MAKE} vendor

.PHONY: build
build: fetch ${BUILD_PATH}/${CMD} ## Build application

# Binary
${BUILD_PATH}/${CMD}: ${GO_FILES} go.sum
	@echo "Building for ${TARGET}..." && \
	mkdir -p ${BUILD_PATH} && \
	CGO_ENABLED=1 go build \
		-mod vendor \
		-trimpath \
		-ldflags "-s -w -X github.com/migz93/wantarr/build.Version=${VERSION} -X github.com/migz93/wantarr/build.GitCommit=${GIT_COMMIT} -X github.com/migz93/wantarr/build.Timestamp=${TIMESTAMP}" \
		-o ${BUILD_PATH}/${CMD} \
		.

.PHONY: install
install: build ## Install binary
	install -m 0755 ${BUILD_PATH}/${CMD} ${DESTDIR}/${CMD}

.PHONY: clean
clean: ## Cleanup
	rm -rf ${DIST_PATH}

.PHONY: fetch
fetch: ## Fetch vendor files
	go mod vendor

.PHONY: release
release: check_goreleaser fetch ## Generate a release, but don't publish
	goreleaser --skip-validate --skip-publish --rm-dist

.PHONY: publish
publish: fetch ## Generate a release, and publish
		docker run --rm --privileged \
			-e GITHUB_TOKEN="${TOKEN}" \
			-e VERSION="${GIT_TAG_NAME}" \
			-e GIT_COMMIT="${GIT_COMMIT}" \
			-e TIMESTAMP="${TIMESTAMP}" \
			-v `pwd`:/go/src/github.com/migz93/wantarr \
			-v /var/run/docker.sock:/var/run/docker.sock \
			-w /go/src/github.com/migz93/wantarr \
			neilotoole/xcgo:latest goreleaser --rm-dist

.PHONY: snapshot
snapshot: fetch ## Generate a snapshot release
	docker run --rm --privileged \
		-e VERSION="${VERSION}" \
		-e GIT_COMMIT="${GIT_COMMIT}" \
		-e TIMESTAMP="${TIMESTAMP}" \
		-v `pwd`:/go/src/github.com/migz93/wantarr \
		-v /var/run/docker.sock:/var/run/docker.sock \
		-w /go/src/github.com/migz93/wantarr \
		neilotoole/xcgo:latest goreleaser --snapshot --skip-validate --skip-publish --rm-dist

.PHONY: help
help:
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'
