# build from root path
FROM golang:1.16 as builder
ARG version="latest"
ARG service_initial

ENV VERSION=$version
ENV SERVICE_INITIAL=$service_initial

COPY . /opt/src
WORKDIR /opt/src

RUN ["make", "api"]

FROM alpine:3.14

ENV TZ="Asia/Shanghai"
RUN apk add --no-cache tzdata

COPY --from=builder /opt/src/build/nocalhost-api /app/nocalhost-api
COPY --from=builder /opt/src/conf/config.prod.yaml.example /app/config/config.yaml

CMD ["/app/nocalhost-api", "-c", "/app/config/config.yaml"]
