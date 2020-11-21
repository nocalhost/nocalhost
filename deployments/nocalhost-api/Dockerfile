# build from root path
# docker build -t nocalhost-api:latest -f ./deployments/nocalhost-api/Dockerfile .
FROM golang as builder

COPY . /opt/src
WORKDIR /opt/src

RUN ["make", "api"]

#FROM ubuntu:20.04
FROM codingcorp-docker.pkg.coding.net/nocalhost/public/minideb:master

RUN ["mkdir", "-p", "/app/config"]
COPY --from=builder /opt/src/nocalhost-api /app/
COPY --from=builder /opt/src/conf/config.prod.yaml.example /app/config/config.yaml

CMD ["/app/nocalhost-api", "-c", "/app/config/config.yaml"]
