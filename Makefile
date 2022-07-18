TARGET_PLATFORMS ?= linux/amd64
IMAGE_REPO ?= openyurt
IMAGE_TAG ?= v0.1.0
GIT_VERSION ?=$(IMAGE_TAG)

DOCKER_USERNAME ?= ""
DOCKER_PASSWD ?= ""

GIT_VERSION ?=$(IMAGE_TAG)

DOCKER_BUILD_ARGS = --build-arg GIT_VERSION=${GIT_VERSION}

ifeq (${REGION}, cn)
DOCKER_BUILD_ARGS += --build-arg GOPROXY=https://goproxy.cn --build-arg MIRROR_REPO=mirrors.aliyun.com
endif

.PHONY: clean all build

all: build

# Build binaries in the host environment
build: fmt vet
	GIT_VERSION=${GIT_VERSION} hack/make-rules/build.sh $(WHAT)

# Run go fmt against code
fmt:
	go fmt ./pkg/... ./cmd/...

# Run go vet against code
vet:
	go vet ./pkg/... ./cmd/...

clean:
	-rm -Rf _output

docker-build:
	hack/make-rules/manifest.sh ${IMAGE_REPO}/edge-proxy:${GIT_VERSION} ${DOCKER_USERNAME} ${DOCKER_PASSWD}
	docker buildx build --push ${DOCKER_BUILD_ARGS} --platform ${TARGET_PLATFORMS} -f hack/dockerfiles/Dockerfile . -t ${IMAGE_REPO}/edge-proxy:${GIT_VERSION}
