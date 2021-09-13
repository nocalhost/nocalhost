FROM nocalhost-docker.pkg.coding.net/nocalhost/public/ubuntu:stable
# 注意容器安全问题
ARG dep_version="latest"
ENV DEP_VERSION=$dep_version
USER root
LABEL maintainer="nocalhost@coding.net"
RUN wget https://github.com/stedolan/jq/releases/download/jq-1.6/jq-linux64 -O /usr/local/bin/jq && chmod +x /usr/local/bin/jq
COPY deployments/dep-install-job/ /nocalhost/
RUN ["chmod", "+x", "/nocalhost/cert.sh", "/nocalhost/installer.sh"]
