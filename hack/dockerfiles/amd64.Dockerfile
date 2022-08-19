FROM --platform=${BUILDPLATFORM} ubuntu:20.04
ARG TARGETOS TARGETARCH MIRROR_REPO
#RUN if [ ! -z "${MIRROR_REPO+x}" ]; then sed -i "s/dl-cdn.alpinelinux.org/${MIRROR_REPO}/g" /etc/apk/repositories; fi && \
#    apk add ca-certificates bash libc6-compat && update-ca-certificates && rm /var/cache/apk/*
RUN apt-get update && apt-get install vim net-tools iproute2 ca-certificates wget  libssl-dev iptables telnet -y
COPY _output/local/bin/${TARGETOS}/${TARGETARCH}/edge-proxy /usr/local/bin/edge-proxy
COPY _output/local/bin/${TARGETOS}/${TARGETARCH}/benchmark /usr/local/bin/benchmark
ENTRYPOINT ["/usr/local/bin/edge-proxy"]
