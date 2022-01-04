BIN := listmonk-messenger.bin

LAST_COMMIT := $(shell git rev-parse --short HEAD)
LAST_COMMIT_DATE := $(shell git show -s --format=%ci ${LAST_COMMIT})
VERSION := $(shell git describe --abbrev=1)
BUILDSTR := ${VERSION} (build "\\\#"${LAST_COMMIT} $(shell date '+%Y-%m-%d %H:%M:%S'))

build:
	go build -o ${BIN} -ldflags="-X 'main.buildString=${BUILDSTR}'" *.go
.PHONY: build

run: build
	@./${BIN}
.PHONY: run

.DEFAULT_GOAL := build
