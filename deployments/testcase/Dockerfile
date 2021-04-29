FROM golang:1.15 as builder
COPY . /opt/src
WORKDIR /opt/src
RUN CGO_ENABLED=0 GOOS=linux go build -o build/testcase nocalhost/test

FROM ubuntu:latest
# install kubectl
RUN sed -i s@/archive.ubuntu.com/@/mirrors.aliyun.com/@g /etc/apt/sources.list
RUN apt-get update && (apt-get install git curl wget -y)
RUN curl -LO "https://dl.k8s.io/release/$(curl -L -s https://dl.k8s.io/release/stable.txt)/bin/linux/amd64/kubectl"
RUN install -o root -g root -m 0755 kubectl /usr/local/bin/kubectl
# install helm
RUN wget https://get.helm.sh/helm-v3.5.3-linux-amd64.tar.gz
RUN (tar -zxvf helm-v3.5.3-linux-amd64.tar.gz) && (mv linux-amd64/helm /usr/local/bin/helm) && (rm helm-v3.5.3-linux-amd64.tar.gz)

COPY --from=builder /opt/src/build/testcase ./main

ENTRYPOINT ["./main"]
