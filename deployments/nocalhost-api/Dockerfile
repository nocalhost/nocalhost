FROM golang as builder

COPY . /opt/src
WORKDIR /opt/src

RUN ["make", "api"]

FROM ubuntu:20.04

RUN ["mkdir", "-p", "/app/config"]
COPY --from=builder /opt/src/nocalhost-api /app/

CMD ["/app/nocalhost-api", "-c", "/app/config/config.yaml"]
