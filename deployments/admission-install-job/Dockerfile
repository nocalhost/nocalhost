From codingcorp-docker.pkg.coding.net/nocalhost/public/kubectl:latest
# 注意容器安全问题
USER root
LABEL maintainer="nocalhost@coding.net"]
RUN wget https://github.com/stedolan/jq/releases/download/jq-1.6/jq-linux64 -O /usr/local/bin/jq && chmod +x /usr/local/bin/jq
COPY . /nocalhost/
RUN ["chmod", "+x", "/nocalhost/cert.sh", "/nocalhost/installer.sh"]