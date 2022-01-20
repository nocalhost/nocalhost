#FROM alpine:latest
#
## set labels for metadata
##LABEL maintainer="CODING DevOps<nocalhost@coding.net>" \
#  name="sidecar-injector" \
#  description="A Kubernetes mutating webhook server that implements sidecar injection" \
#  summary="A Kubernetes mutating webhook server that implements sidecar injection"
#
## set environment variables
##ENV SIDECAR_INJECTOR=/usr/local/bin/sidecar-injector \
##  USER_UID=1001 \
##  USER_NAME=sidecar-injector
#
## install sidecar-injector binary
#COPY build/_output/bin/sidecar-injector /usr/local/bin/sidecar-injector
#
## copy licenses
##RUN mkdir /licenses
##COPY LICENSE /licenses
#
## set entrypoint
#ENTRYPOINT ["./usr/local/bin/sidecar-injector"]
#
## switch to non-root user
##USER ${USER_UID}

# 从项目根目录 build ：docker build -t nocalhost-admission-webhook:v51 -f deployments/dep-install-job/webhook/Dockerfile .

# Compile stage
FROM golang:1.16.7 AS build-env

# Build Delve
#RUN go get github.com/go-delve/delve/cmd/dlv

COPY . /dockerdev
WORKDIR /dockerdev/cmd/nocalhost-dep

#RUN go build -gcflags="all=-N -l" -o /server
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -gcflags="all=-N -l" -o /server

# Final stage
# FROM debian:buster
FROM nocalhost-docker.pkg.coding.net/nocalhost/public/ubuntu:stable

WORKDIR /
#COPY --from=build-env /go/bin/dlv /
COPY --from=build-env /server /
# COPY --from=build-env /dockerdev /dockerdev

# RUN apt-get update && apt-get install -y inotify-tools

# ENTRYPOINT sh /dockerdev/deployment/startScript.sh
# CMD ["/dlv", "--listen=:8443", "--headless=true", "--api-version=2", "--accept-multiclient", "exec", "/server"]
ENTRYPOINT ["/server"]
