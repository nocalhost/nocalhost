FROM alpine:3.7

MAINTAINER CODING DevOps <nocalhost@coding.net>

ENV KUBE_LATEST_VERSION="v1.18.1"

RUN apk add --update ca-certificates curl jq \
 && curl -L https://storage.googleapis.com/kubernetes-release/release/${KUBE_LATEST_VERSION}/bin/linux/amd64/kubectl -o /usr/local/bin/kubectl \
 && chmod +x /usr/local/bin/kubectl \
 && rm /var/cache/apk/*

ADD wait_for.sh /usr/local/bin/wait_for.sh

RUN chmod +x /usr/local/bin/wait_for.sh

ENTRYPOINT ["/usr/local/bin/wait_for.sh"]