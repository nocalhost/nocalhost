# build from root path
FROM golang as builder
ARG version="latest"
ARG service_initial

ENV VERSION=$version
ENV SERVICE_INITIAL=$service_initial

COPY . /opt/src
WORKDIR /opt/src

RUN ["make", "api"]

FROM nocalhost-docker.pkg.coding.net/nocalhost/public/base-debian11:distroless

ENV TZ="Asia/Shanghai"

COPY --from=builder /opt/src/build/nocalhost-api /app/nocalhost-api
COPY --from=builder /opt/src/conf/config.prod.yaml.example /app/config/config.yaml

CMD ["/app/nocalhost-api", "-c", "/app/config/config.yaml"]
