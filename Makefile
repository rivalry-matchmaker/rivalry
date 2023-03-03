# VARIABLES
# =====================
# the root dir of the repo, this should be overwritten by the Makefile that imports this
REPOSITORY_ROOT := $(patsubst %/,%,$(dir $(abspath $(MAKEFILE_LIST))))

# location used for build output
BUILD_DIR = $(REPOSITORY_ROOT)/build

# location for storing build tools
TOOLCHAIN_DIR = $(BUILD_DIR)/toolchain

# location for storing build tool binaries
TOOLCHAIN_BIN = $(TOOLCHAIN_DIR)/bin

# windows .exe support
EXE_EXTENSION =

# the version of google api we are working with
GOOGLE_APIS_VERSION = aba342359b6743353195ca53f944fe71e6fb6cd4

# the golang command (wrapped to turn on go module support)
GO = GO111MODULE=on go

REGISTRY ?=
TAG ?= latest

# SUBCOMMANDS
# =====================
# build toolchain bin folder
$(TOOLCHAIN_BIN):
	mkdir -p $(TOOLCHAIN_BIN)

# COMMANDS
# =====================
check: clean go-fmt go-vet go-test go-lint

clean:
	rm -rf $(BUILD_DIR)/

go-fmt:
	find . -iname '*go' | xargs gofmt -w

go-vet:
	$(GO) vet ./...

go-test:
	mkdir -p $(BUILD_DIR)/coverage/
	$(GO) test -failfast -v -coverprofile=$(BUILD_DIR)/coverage/coverage.out ./...
	$(GO) tool cover -func=$(BUILD_DIR)/coverage/coverage.out

go-lint:
	go install golang.org/x/lint/golint
	golint -set_exit_status ./...

coverage: go-test
	$(GO) tool cover -html=$(BUILD_DIR)/coverage/coverage.out

build-frontend:
	docker build -f Dockerfile.cmd --build-arg=COMMAND=frontend -t $(REGISTRY)rivalry-frontend:$(TAG) .

build-accumulator:
	docker build -f Dockerfile.cmd --build-arg=COMMAND=accumulator -t $(REGISTRY)rivalry-accumulator:$(TAG) .

build-matcher:
	docker build -f Dockerfile.cmd --build-arg=COMMAND=matcher -t $(REGISTRY)rivalry-matcher:$(TAG) .

build-dispenser:
	docker build -f Dockerfile.cmd --build-arg=COMMAND=dispenser -t $(REGISTRY)rivalry-dispenser:$(TAG) .

build-matchmaker:
	docker build -f Dockerfile.cmd --build-arg=LOCATION=examples --build-arg=COMMAND=matchmaker \
		-t $(REGISTRY)rivalry-matchmaker:$(TAG) .

build-assignment:
	docker build -f Dockerfile.cmd --build-arg=LOCATION=examples --build-arg=COMMAND=assignment \
		-t $(REGISTRY)rivalry-assignment:$(TAG) .

build: build-frontend build-accumulator build-matcher build-dispenser build-matchmaker build-assignment

build-k6: $(TOOLCHAIN_BIN)
	go install go.k6.io/xk6/cmd/xk6@latest
	xk6 build --with github.com/rivalry-matchmaker/rivalry/xk6/xk6-frontend=$(REPOSITORY_ROOT)/xk6/xk6-frontend --output $(TOOLCHAIN_BIN)/k6

run-k6: build-k6
	$(TOOLCHAIN_BIN)/k6 run xk6/test.js
