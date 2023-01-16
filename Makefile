# VARIABLES
# =====================
# the root dir of the repo, this should be overwritten by the Makefile that imports this
REPOSITORY_ROOT := $(patsubst %/,%,$(dir $(abspath $(MAKEFILE_LIST))))

# list of protobuf files that need building
GOLANG_PROTOS = pkg/pb/frontend.pb.go pkg/pb/frontend.pb.gw.go api/frontend.swagger.json \
	pkg/pb/messages.pb.go pkg/pb/internal.pb.go pkg/pb/match_logic.pb.go

# the version of protoc used to build protobuf
PROTOC_VERSION = 3.19.1

# location used for build output
BUILD_DIR = $(REPOSITORY_ROOT)/build

# location for storing build tools
TOOLCHAIN_DIR = $(BUILD_DIR)/toolchain

# location for storing build tool binaries
TOOLCHAIN_BIN = $(TOOLCHAIN_DIR)/bin

# location for storing third party protobuf files, for use while building
PROTOC_INCLUDES := $(REPOSITORY_ROOT)/third_party

# windows .exe support
EXE_EXTENSION =

# wrapper for protobuf command
PROTOC := PATH=$(TOOLCHAIN_BIN) $(TOOLCHAIN_BIN)/protoc$(EXE_EXTENSION)

# the version of google api we are working with
GOOGLE_APIS_VERSION = aba342359b6743353195ca53f944fe71e6fb6cd4

# the golang command (wrapped to turn on go module support)
GO = GO111MODULE=on go

REGISTRY ?=
TAG ?= latest

# Platform specific requirements
ifeq ($(OS),Windows_NT)
	EXE_EXTENSION = .exe
	PROTOC_PACKAGE = https://github.com/protocolbuffers/protobuf/releases/download/v$(PROTOC_VERSION)/protoc-$(PROTOC_VERSION)-win64.zip
else
	UNAME_S := $(shell uname -s)
	ifeq ($(UNAME_S),Linux)
		PROTOC_PACKAGE = https://github.com/protocolbuffers/protobuf/releases/download/v$(PROTOC_VERSION)/protoc-$(PROTOC_VERSION)-linux-x86_64.zip
	endif
	ifeq ($(UNAME_S),Darwin)
		PROTOC_PACKAGE = https://github.com/protocolbuffers/protobuf/releases/download/v$(PROTOC_VERSION)/protoc-$(PROTOC_VERSION)-osx-x86_64.zip
	endif
endif

# SUBCOMMANDS
# =====================
# build toolchain bin folder
$(TOOLCHAIN_BIN):
	mkdir -p $(TOOLCHAIN_BIN)

# add protoc plugins
$(TOOLCHAIN_BIN)/protoc-gen-%$(EXE_EXTENSION): $(TOOLCHAIN_BIN)
	GOBIN=$(TOOLCHAIN_BIN) $(GO) install \
		github.com/golang/protobuf/protoc-gen-go \
		github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-grpc-gateway \
		github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-openapiv2

# build protoc
build/toolchain/bin/protoc$(EXE_EXTENSION): $(TOOLCHAIN_BIN)/protoc-gen-%$(EXE_EXTENSION)
	curl -o $(TOOLCHAIN_DIR)/protoc-temp.zip -L $(PROTOC_PACKAGE)
	(cd $(TOOLCHAIN_DIR); unzip -q -o protoc-temp.zip)
	rm $(TOOLCHAIN_DIR)/protoc-temp.zip $(TOOLCHAIN_DIR)/readme.txt

# build swagger json
api/%.swagger.json: api/%.proto build/toolchain/bin/protoc$(EXE_EXTENSION) third_party
	mkdir -p $(REPOSITORY_ROOT)/build/prototmp $(REPOSITORY_ROOT)/pkg/pb
	$(PROTOC) $< \
		-I $(REPOSITORY_ROOT) \
		-I $(PROTOC_INCLUDES) \
		--openapiv2_out=json_names_for_fields=true,logtostderr=true:$(REPOSITORY_ROOT)/build/prototmp
	mv $(REPOSITORY_ROOT)/build/prototmp/api/$(@F) $@

# build grpc gateway
pkg/pb/%.pb.gw.go: api/%.proto build/toolchain/bin/protoc$(EXE_EXTENSION) third_party
	mkdir -p $(REPOSITORY_ROOT)/build/prototmp $(REPOSITORY_ROOT)/pkg/pb
	$(PROTOC) $< \
		-I $(REPOSITORY_ROOT) \
		-I $(PROTOC_INCLUDES) \
		--grpc-gateway_out=logtostderr=true:$(REPOSITORY_ROOT)/build/prototmp
	mv $(REPOSITORY_ROOT)/build/prototmp/github.com/rivalry-matchmaker/rivalry/$@ $@

# build grpc boilerplate
pkg/pb/%.pb.go: api/%.proto build/toolchain/bin/protoc$(EXE_EXTENSION) third_party
	mkdir -p $(REPOSITORY_ROOT)/build/prototmp $(REPOSITORY_ROOT)/pkg/pb
	PATH=$(TOOLCHAIN_BIN) $(PROTOC) $< \
		-I $(REPOSITORY_ROOT) -I $(PROTOC_INCLUDES) \
		--go_out=plugins=grpc:$(REPOSITORY_ROOT)/build/prototmp
	mv $(REPOSITORY_ROOT)/build/prototmp/github.com/rivalry-matchmaker/rivalry/$@ $@

third_party: third_party/google/api third_party/protoc-gen-openapiv2/options

# acquire google api third party library
third_party/google/api:
	mkdir -p $(TOOLCHAIN_DIR)/googleapis-temp/
	mkdir -p $(REPOSITORY_ROOT)/third_party/google/api
	mkdir -p $(REPOSITORY_ROOT)/third_party/google/rpc
	curl -o $(TOOLCHAIN_DIR)/googleapis-temp/googleapis.zip -L https://github.com/googleapis/googleapis/archive/$(GOOGLE_APIS_VERSION).zip
	(cd $(TOOLCHAIN_DIR)/googleapis-temp/; unzip -q -o googleapis.zip)
	cp -f $(TOOLCHAIN_DIR)/googleapis-temp/googleapis-$(GOOGLE_APIS_VERSION)/google/api/*.proto $(REPOSITORY_ROOT)/third_party/google/api/
	cp -f $(TOOLCHAIN_DIR)/googleapis-temp/googleapis-$(GOOGLE_APIS_VERSION)/google/rpc/*.proto $(REPOSITORY_ROOT)/third_party/google/rpc/
	rm -rf $(TOOLCHAIN_DIR)/googleapis-temp

# acquire protoc-gen-openapiv2 library
third_party/protoc-gen-openapiv2/options:
	mkdir -p $(TOOLCHAIN_DIR)/grpc-gateway-temp/
	mkdir -p $(REPOSITORY_ROOT)/third_party/protoc-gen-openapiv2/options
	curl -o $(TOOLCHAIN_DIR)/grpc-gateway-temp/grpc-gateway.zip -L https://github.com/grpc-ecosystem/grpc-gateway/archive/v$(GRPC_GATEWAY_VERSION).zip
	(cd $(TOOLCHAIN_DIR)/grpc-gateway-temp/; unzip -q -o grpc-gateway.zip)
	cp -f $(TOOLCHAIN_DIR)/grpc-gateway-temp/grpc-gateway-$(GRPC_GATEWAY_VERSION)/protoc-gen-openapiv2/options/*.proto $(REPOSITORY_ROOT)/third_party/protoc-gen-openapiv2/options/
	rm -rf $(TOOLCHAIN_DIR)/grpc-gateway-temp

# COMMANDS
# =====================
protobuf: $(GOLANG_PROTOS)

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

build-dispenser:
	docker build -f Dockerfile.cmd --build-arg=COMMAND=dispenser -t $(REGISTRY)rivalry-dispenser:$(TAG) .

build-matchmaker:
	docker build -f Dockerfile.cmd --build-arg=LOCATION=examples --build-arg=COMMAND=matchmaker \
		-t $(REGISTRY)rivalry-matchmaker:$(TAG) .

build-assignment:
	docker build -f Dockerfile.cmd --build-arg=LOCATION=examples --build-arg=COMMAND=assignment \
		-t $(REGISTRY)rivalry-assignment:$(TAG) .

build: build-frontend build-accumulator build-dispenser build-matchmaker build-assignment

build-k6: $(TOOLCHAIN_BIN)
	go install go.k6.io/xk6/cmd/xk6@latest
	xk6 build --with github.com/rivalry-matchmaker/rivalry/xk6/xk6-frontend=$(REPOSITORY_ROOT)/xk6/xk6-frontend --output $(TOOLCHAIN_BIN)/k6

run-k6: build-k6
	$(TOOLCHAIN_BIN)/k6 run xk6/test.js
