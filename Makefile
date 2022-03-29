#
# Copyright 2021 The Kubernetes Authors.

# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at

#    http://www.apache.org/licenses/LICENSE-2.0

# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
##

EXE_DRIVER_NAME=ibm-vpc-block-csi-driver
DRIVER_NAME=vpcBlockDriver
GIT_COMMIT_SHA="$(shell git rev-parse HEAD 2>/dev/null)"
GIT_REMOTE_URL="$(shell git config --get remote.origin.url 2>/dev/null)"
BUILD_DATE="$(shell date -u +"%Y-%m-%dT%H:%M:%SZ")"
OSS_FILES := go.mod Dockerfile
GOLANG_VERSION="1.16.13"


STAGING_REGISTRY ?= gcr.io/k8s-staging-cloud-provider-ibm
REGISTRY ?= $(STAGING_REGISTRY)
RELEASE_TAG ?= $(shell git describe --abbrev=0 2>/dev/null)
PULL_BASE_REF ?= $(RELEASE_TAG) # PULL_BASE_REF will be provided by Prow
RELEASE_ALIAS_TAG ?= $(PULL_BASE_REF)

CORE_IMAGE_NAME ?= $(EXE_DRIVER_NAME)
CORE_DRIVER_IMG ?= $(REGISTRY)/$(CORE_IMAGE_NAME)

TAG ?= dev
ARCH ?= amd64
ALL_ARCH ?= amd64 ppc64le




# Jenkins vars. Set to `unknown` if the variable is not yet defined
BUILD_NUMBER?=unknown
GO111MODULE_FLAG?=on
export GO111MODULE=$(GO111MODULE_FLAG)

export LINT_VERSION="1.31.0"

COLOR_YELLOW=\033[0;33m
COLOR_RESET=\033[0m

.PHONY: all
all: deps fmt build test buildimage

.PHONY: driver
driver: deps buildimage

.PHONY: deps
deps:
	echo "Installing dependencies ..."
	@if ! which golangci-lint >/dev/null || [[ "$$(golangci-lint --version)" != *${LINT_VERSION}* ]]; then \
		curl -sfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(shell go env GOPATH)/bin v${LINT_VERSION}; \
	fi

.PHONY: fmt
fmt: lint
	$(GOPATH)/bin/golangci-lint run --disable-all --enable=gofmt --timeout 600s
	@if [ -n "$$($(GOPATH)/bin/golangci-lint run)" ]; then echo 'Please run ${COLOR_YELLOW}make dofmt${COLOR_RESET} on your code.' && exit 1; fi

.PHONY: dofmt
dofmt:
	$(GOPATH)/bin/golangci-lint run --disable-all --enable=gofmt --fix --timeout 600s

.PHONY: lint
lint:
	$(GOPATH)/bin/golangci-lint run --timeout 600s

.PHONY: build
build:
	CGO_ENABLED=0 GOOS=$(shell go env GOOS) GOARCH=$(shell go env GOARCH) go build -mod=vendor -a -ldflags '-X main.vendorVersion='"${DRIVER_NAME}-${GIT_COMMIT_SHA}"' -extldflags "-static"' -o ${GOPATH}/bin/${EXE_DRIVER_NAME} ./cmd/

.PHONY: verify
verify:
	echo "Verifying and linting files ..."
	./hack/verify-all.sh
	echo "Congratulations! All Go source files have been linted."

.PHONY: test
test:
	go test -v -race ./cmd/... ./pkg/...

.PHONY: ut-coverage
ut-coverage:
	go tool cover -html=cover.out -o=cover.html

.PHONY: buildimage
buildimage: build-systemutil
	docker build	\
        --build-arg git_commit_id=${GIT_COMMIT_SHA} \
        --build-arg git_remote_url=${GIT_REMOTE_URL} \
        --build-arg build_date=${BUILD_DATE} \
        --build-arg jenkins_build_number=${BUILD_NUMBER} \
        --build-arg REPO_SOURCE_URL=${REPO_SOURCE_URL} \
        --build-arg BUILD_URL=${BUILD_URL} \
	-t $(CORE_DRIVER_IMG):$(ARCH)-$(TAG) -f Dockerfile .

.PHONY: build-systemutil
build-systemutil:
	docker build --build-arg TAG=$(GIT_COMMIT_SHA) --build-arg OS=linux --build-arg ARCH=$(ARCH) -t csi-driver-builder --pull -f Dockerfile.builder .
	docker run --env GHE_TOKEN=${GHE_TOKEN} csi-driver-builder
	docker cp `docker ps -q -n=1`:/go/bin/${EXE_DRIVER_NAME} ./${EXE_DRIVER_NAME}

.PHONY: test-sanity
test-sanity: deps fmt
	SANITY_PARAMS_FILE=./csi_sanity_params.yaml go test -timeout 160s ./tests/sanity -run ^TestSanity$$ -v

.PHONY: clean
clean:
	rm -rf ${EXE_DRIVER_NAME}
	rm -rf $(GOPATH)/bin/${EXE_DRIVER_NAME}

## --------------------------------------
## Docker
## --------------------------------------

.PHONY: docker-build
docker-build: docker-pull-prerequisites buildimage ## Build the docker image for ibm-vpc-block-csi-driver

.PHONY: docker-push
docker-push: ## Push the docker image
	docker push $(CORE_DRIVER_IMG):$(ARCH)-$(TAG)

.PHONY: docker-pull-prerequisites
docker-pull-prerequisites:
	docker pull docker.io/docker/dockerfile:1.1-experimental
	docker pull docker.io/library/golang:$(GOLANG_VERSION)
	docker pull gcr.io/distroless/static:latest


## --------------------------------------
## Docker - All ARCH
## --------------------------------------

.PHONY: docker-build-all ## Build all the architecture docker images
docker-build-all: $(addprefix docker-build-,$(ALL_ARCH))

docker-build-%:
	$(MAKE) ARCH=$* docker-build

.PHONY: docker-push-all ## Push all the architecture docker images
docker-push-all: $(addprefix docker-push-,$(ALL_ARCH))
	$(MAKE) docker-push-core-manifest

docker-push-%:
	$(MAKE) ARCH=$* docker-push

.PHONY: docker-push-core-manifest
docker-push-core-manifest: ## Push the fat manifest docker image.
	## Minimum docker version 18.06.0 is required for creating and pushing manifest images.
	$(MAKE) docker-push-manifest DRIVER_IMG=$(CORE_DRIVER_IMG) MANIFEST_FILE=$(CORE_MANIFEST_FILE)

.PHONY: docker-push-manifest
docker-push-manifest:
	docker manifest create --amend $(DRIVER_IMG):$(TAG) $(shell echo $(ALL_ARCH) | sed -e "s~[^ ]*~$(DRIVER_IMG):&-$(TAG)~g")
	@for arch in $(ALL_ARCH); do docker manifest annotate --arch $${arch} ${DRIVER_IMG}:${TAG} ${DRIVER_IMG}:$${arch}-${TAG}; done
	docker manifest push --purge ${DRIVER_IMG}:${TAG}

## --------------------------------------
## Release
## --------------------------------------

.PHONY: release-alias-tag
release-alias-tag: # Adds the tag to the last build tag.
	#gcloud container images add-tag -q $(CORE_DRIVER_IMG):$(TAG) $(CORE_DRIVER_IMG):$(RELEASE_ALIAS_TAG)
	docker pull $(CORE_DRIVER_IMG):$(TAG)
	docker tag $(CORE_DRIVER_IMG):$(TAG) $(CORE_DRIVER_IMG):$(RELEASE_ALIAS_TAG)
	docker push $(CORE_DRIVER_IMG):$(RELEASE_ALIAS_TAG)

.PHONY: release-staging
release-staging: ## Builds and push container images to the staging image registry.
	$(MAKE) docker-build-all
	$(MAKE) docker-push-all
	$(MAKE) release-alias-tag
