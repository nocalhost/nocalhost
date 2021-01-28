# build from root path
FROM golang as builder
ARG version="latest"
ENV VERSION=$version

COPY . /opt/src
WORKDIR /opt/src

RUN ["make", "api"]

#FROM ubuntu:20.04
FROM codingcorp-docker.pkg.coding.net/nocalhost/public/ubuntu:stable

RUN ln -fs /usr/share/zoneinfo/Asia/Shanghai /etc/localtime
RUN apt-get install -y tzdata

RUN ["mkdir", "-p", "/app/config"]
COPY --from=builder /opt/src/build/nocalhost-api /app/nocalhost-api
COPY --from=builder /opt/src/conf/config.prod.yaml.example /app/config/config.yaml

CMD ["/app/nocalhost-api", "-c", "/app/config/config.yaml"]
