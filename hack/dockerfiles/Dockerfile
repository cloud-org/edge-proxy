# multi-arch image building for edge proxy

FROM --platform=${BUILDPLATFORM} golang:1.17.1 as builder
ADD . /build
ARG TARGETOS TARGETARCH GIT_VERSION GOPROXY MIRROR_REPO
WORKDIR /build/
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} GIT_VERSION=${GIT_VERSION} make build

#FROM --platform=${TARGETPLATFORM} alpine:3.14
FROM --platform=${BUILDPLATFORM} ubuntu:20.04
ARG TARGETOS TARGETARCH MIRROR_REPO
#RUN if [ ! -z "${MIRROR_REPO+x}" ]; then sed -i "s/dl-cdn.alpinelinux.org/${MIRROR_REPO}/g" /etc/apk/repositories; fi && \
#    apk add ca-certificates bash libc6-compat && update-ca-certificates && rm /var/cache/apk/*
RUN apt-get update && apt-get install vim net-tools iproute2 ca-certificates wget  libssl-dev iptables telnet -y
COPY --from=builder /build/_output/local/bin/${TARGETOS}/${TARGETARCH}/edge-proxy /usr/local/bin/edge-proxy
COPY --from=builder /build/_output/local/bin/${TARGETOS}/${TARGETARCH}/benchmark /usr/local/bin/benchmark
ENTRYPOINT ["/usr/local/bin/edge-proxy"]
