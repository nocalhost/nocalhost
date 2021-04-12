FROM golang:1.15 as builder
COPY . /opt/src
WORKDIR /opt/src
RUN CGO_ENABLED=0 GOOS=linux go build -mod=readonly -o build/testcase nocalhost/test

FROM ubuntu:latest
# install kubectl
RUN sed -i s@/archive.ubuntu.com/@/mirrors.aliyun.com/@g /etc/apt/sources.list
RUN apt-get update && (apt-get install git apt-transport-https gnupg2 curl wget -y)
RUN (curl -s https://packages.cloud.google.com/apt/doc/apt-key.gpg | apt-key add -)
RUN echo "deb https://apt.kubernetes.io/ kubernetes-xenial main" | tee -a /etc/apt/sources.list.d/kubernetes.list
RUN (apt-get update) && (apt-get install -y kubectl)
# install helm
RUN wget https://get.helm.sh/helm-v3.5.3-linux-amd64.tar.gz
RUN (tar -zxvf helm-v3.5.3-linux-amd64.tar.gz) && (mv linux-amd64/helm /usr/local/bin/helm) && (rm helm-v3.5.3-linux-amd64.tar.gz)

COPY --from=builder /opt/src/build/testcase ./main

ENTRYPOINT ["./main"]