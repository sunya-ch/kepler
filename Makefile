### env define ###
export BIN_TIMESTAMP ?=$(shell date -u +'%Y-%m-%dT%H:%M:%SZ')
export TIMESTAMP ?=$(shell echo $(BIN_TIMESTAMP) | tr -d ':' | tr 'T' '-' | tr -d 'Z')

# restrict included verify-* targets to only process project files
SOURCE_GIT_TAG :=$(shell git describe --tags --always --abbrev=7 --match 'v*')
SRC_ROOT :=$(shell pwd)
OUTPUT_DIR :=_output
CROSS_BUILD_BINDIR :=$(OUTPUT_DIR)/bin
GIT_VERSION     := $(shell git describe --dirty --tags --always --match='v*')
VERSION         ?= $(GIT_VERSION)
LDFLAGS         := "-w -s -X 'github.com/sustainable-computing-io/kepler/pkg/version.Version=$(VERSION)'"
ROOTLESS	?= false

ifdef IMAGE_REPO
	IMAGE_REPO := $(IMAGE_REPO)
else
	IMAGE_REPO := quay.io/sustainable_computing_io
endif

ifdef BUILDER_IMAGE
	BUILDER_IMAGE := $(BUILD_IMAGE)
else
	BUILDER_IMAGE := quay.io/sustainable_computing_io/kepler_builder:ubi-8.6-bcc-0.24-go1.18
endif

ifdef IMAGE_TAG
	IMAGE_TAG := $(IMAGE_TAG)
else
	IMAGE_TAG := latest
endif

ifdef CTR_CMD
	CTR_CMD := $(CTR_CMD)
else 
	CTR_CMD :=$(or $(shell which podman 2>/dev/null), $(shell which docker 2>/dev/null))
endif


ifeq ($(DEBUG),true)
	# throw all the debug info in!
	LD_FLAGS =
	GC_FLAGS =-gcflags "all=-N -l"
else
	# strip everything we can
	LD_FLAGS =-w -s
	GC_FLAGS =
endif

GO_LD_FLAGS := $(GC_FLAGS) -ldflags "-X $(LD_FLAGS)" $(CFLAGS)

GO_BUILD_TAGS := 'include_gcs include_oss containers_image_openpgp gssapi providerless netgo osusergo gpu'
ifneq ($(shell command -v ldconfig),)
  ifneq ($(shell ldconfig -p|grep bcc),)
     GO_BUILD_TAGS = 'include_gcs include_oss containers_image_openpgp gssapi providerless netgo osusergo gpu bcc'
  endif
endif

ifneq ($(shell command -v dpkg),)
	ifneq ($(shell dpkg -l|grep bcc),)
		GO_BUILD_TAGS = 'include_gcs include_oss containers_image_openpgp gssapi providerless netgo osusergo gpu bcc'
	endif
endif

GOOS := $(shell go env GOOS)
GOARCH := $(shell go env GOARCH)

# for testsuite
ENVTEST_ASSETS_DIR=./test-bin
export PATH := $(PATH):./test-bin

ifndef GOPATH
  GOPATH := $(HOME)/go
  GOBIN := $(GOPATH)/bin
endif

KUSTOMIZE = $(shell pwd)/bin/kustomize

# go-get-tool will 'go get' any package $2 and install it to $1.
PROJECT_DIR := $(shell dirname $(abspath $(firstword $(MAKEFILE_LIST))))
define go-get-tool
@[ -f $(1) ] || { \
set -e ;\
TMP_DIR=$$(mktemp -d) ;\
cd $$TMP_DIR ;\
go mod init tmp ;\
echo "Downloading $(2)" ;\
GOBIN=$(PROJECT_DIR)/bin go install $(2) ;\
ls $$TMP_DIR;\
echo $(PROJECT_DIR);\
rm -rf $$TMP_DIR ;\
}
endef

base_dir := $(patsubst %/,%,$(dir $(realpath $(lastword $(MAKEFILE_LIST)))))

### Default ###
kepler: build_containerized
.PHONY: kepler

clean: clean-cross-build
.PHONY: clean

### build container ###
build_containerized: genbpfassets tidy-vendor format
	@if [ -z '$(CTR_CMD)' ] ; then echo '!! ERROR: containerized builds require podman||docker CLI, none found $$PATH' >&2 && exit 1; fi
	echo BIN_TIMESTAMP==$(BIN_TIMESTAMP)

	$(CTR_CMD) build -t $(IMAGE_REPO)/kepler:$(SOURCE_GIT_TAG)-linux-$(GOARCH) \
		-f "$(SRC_ROOT)"/build/Dockerfile \
		--build-arg SOURCE_GIT_TAG=$(SOURCE_GIT_TAG) \
		--build-arg BIN_TIMESTAMP=$(BIN_TIMESTAMP) \
		--platform="linux/$(GOARCH)" \
		.

	$(CTR_CMD) tag $(IMAGE_REPO)/kepler:$(SOURCE_GIT_TAG)-linux-$(GOARCH) $(IMAGE_REPO)/kepler:$(IMAGE_TAG)

.PHONY: build_containerized

push-image:
	$(CTR_CMD) push $(IMAGE_REPO)/kepler:$(IMAGE_TAG)
.PHONY: push-image

clean-cross-build:
	$(RM) -r '$(CROSS_BUILD_BINDIR)'
	$(RM) -rf $(OUTPUT_DIR)/staging
	if [ -d '$(OUTPUT_DIR)' ]; then rmdir --ignore-fail-on-non-empty '$(OUTPUT_DIR)'; fi
.PHONY: clean-cross-build

### build binary ###
build: clean_build_local _build_local copy_build_local
.PHONY: build

_build_local: tidy-vendor format
	@mkdir -p "$(CROSS_BUILD_BINDIR)/$(GOOS)_$(GOARCH)"
	+@GOOS=$(GOOS) GOARCH=$(GOARCH) go build -v -tags ${GO_BUILD_TAGS} \
		-o $(CROSS_BUILD_BINDIR)/$(GOOS)_$(GOARCH)/kepler -ldflags $(LDFLAGS) ./cmd/exporter.go

container_build: tidy-vendor format
	$(CTR_CMD) run --rm \
		-v $(base_dir):/kepler:Z -w /kepler \
		-e GOROOT=/usr/local/go -e PATH=$(PATH):/usr/local/go/bin \
		$(BUILDER_IMAGE) \
		make build

build_rpm:
	rpmbuild packaging/rpm/kepler.spec --build-in-place -bb

build_container_rpm:
	rpmbuild packaging/rpm/container-kepler.spec --build-in-place -bb

containerized_build_rpm:
	@mkdir -p $(base_dir)/$(OUTPUT_DIR)/rpmbuild
	$(CTR_CMD) run --rm \
		-v $(base_dir):/kepler:Z -w /kepler -v $(base_dir)/$(OUTPUT_DIR)/rpmbuild:/root/rpmbuild \
		-e _VERSION_=${_VERSION_} -e _RELEASE_=${_RELEASE_} -e _ARCH_=${_ARCH_} \
		-e _TIMESTAMP_="$(shell date)" -e _COMMITTER_=${_COMMITTER_} -e  _CHANGELOG_=${_CHANGELOG_} \
		-e GOROOT=/usr/local/go -e PATH=$(PATH):/usr/local/go/bin \
		$(BUILDER_IMAGE) \
		make build_rpm

containerized_build_container_rpm:
	@mkdir -p $(base_dir)/$(OUTPUT_DIR)/rpmbuild
	$(CTR_CMD) run --rm \
		-v $(base_dir):/kepler:Z -w /kepler -v $(base_dir)/$(OUTPUT_DIR)/rpmbuild:/root/rpmbuild \
		-e _VERSION_=${_VERSION_} -e _RELEASE_=${_RELEASE_} \
		$(BUILDER_IMAGE) \
		make build_container_rpm

clean_build_local:
	rm -rf $(CROSS_BUILD_BINDIR)

copy_build_local:
	cp $(CROSS_BUILD_BINDIR)/$(GOOS)_$(GOARCH)/kepler $(CROSS_BUILD_BINDIR)

cross-build-linux-amd64: genbpfassets
	+$(MAKE) _build_local GOOS=linux GOARCH=amd64
.PHONY: cross-build-linux-amd64

cross-build-linux-arm64: genbpfassets
	+$(MAKE) _build_local GOOS=linux GOARCH=arm64
.PHONY: cross-build-linux-arm64

cross-build-linux-s390x: genbpfassets
	+$(MAKE) _build_local GOOS=linux GOARCH=s390x
.PHONY: cross-build-linux-s390x

cross-build: clean_build_local cross-build-linux-amd64 cross-build-linux-arm64 cross-build-linux-s390x copy_build_local
.PHONY: cross-build

### toolkit ###
tidy-vendor:
	go mod tidy
	go mod vendor

ginkgo-set: tidy-vendor
	mkdir -p $(GOBIN)
	mkdir -p ${ENVTEST_ASSETS_DIR}
	@test -f $(ENVTEST_ASSETS_DIR)/ginkgo || \
	 (go install -mod=mod github.com/onsi/ginkgo/v2/ginkgo@v2.4.0  && \
	  cp $(GOBIN)/ginkgo $(ENVTEST_ASSETS_DIR)/ginkgo)

test: ginkgo-set tidy-vendor
	@go test -tags $(GO_BUILD_TAGS) ./... --race --bench=. -cover --count=1 --vet=all

test-verbose: ginkgo-set tidy-vendor
	@go test -tags $(GO_BUILD_TAGS) -covermode=atomic -coverprofile=coverage.out -v $$(go list ./... | grep pkg | grep -v bpfassets) --race --bench=. -cover --count=1 --vet=all

test-mac-verbose: ginkgo-set tidy-vendor
	@go test ./... --race --bench=. -cover --count=1 --vet=all

escapes_detect: tidy-vendor
	@go build -tags $(GO_BUILD_TAGS) -gcflags="-m -l" ./... 2>&1 | grep "escapes to heap" || true

set_govulncheck:
	@go install golang.org/x/vuln/cmd/govulncheck@latest

govulncheck: set_govulncheck tidy-vendor
	@govulncheck -v ./... || true

format:
	./automation/presubmit-tests/gofmt.sh

golint:
	$(CTR_CMD) pull golangci/golangci-lint:latest
	$(CTR_CMD) run --tty --rm \
		--volume '$(base_dir)/.cache/golangci-lint:/root/.cache' \
		--volume '$(base_dir):/app' \
		--workdir /app \
		golangci/golangci-lint \
		golangci-lint run --verbose

genbpfassets:
	GO111MODULE=off go get -d github.com/go-bindata/go-bindata/...
	./hack/bindata.sh
.PHONY: genbpfassets

### k8s ###
kustomize: ## Download kustomize locally if necessary.
	mkdir -p bin
	$(call go-get-tool,$(KUSTOMIZE),sigs.k8s.io/kustomize/kustomize/v4@v4.5.2)

build-manifest: kustomize
	./hack/build-manifest.sh "${OPTS}"
.PHONY: build-manifest

cluster-clean: build-manifest
	./hack/cluster-clean.sh
.PHONY: cluster-clean

cluster-deploy: cluster-clean
	BARE_METAL_NODE_ONLY=false ./hack/cluster-deploy.sh
.PHONY: cluster-deploy

cluster-sync:
	./hack/cluster-sync.sh
.PHONY: cluster-sync

cluster-up:
	rm -rf local-dev-cluster
	git clone https://github.com/sustainable-computing-io/local-dev-cluster.git --depth=1
	cd local-dev-cluster && ./main.sh
.PHONY: cluster-up

