# Nocalhost

![api-docker](https://github.com/nocalhost/nocalhost/workflows/api-docker/badge.svg?branch=main)
![nhctl](https://github.com/nocalhost/nocalhost/workflows/nhctl/badge.svg?branch=main)
[![Go Report Card](https://goreportcard.com/badge/github.com/nocalhost/nocalhost)](https://goreportcard.com/report/github.com/nocalhost/nocalhost)
[![LICENSE](https://img.shields.io/github/license/nocalhost/nocalhost)](https://github.com/nocalhost/nocalhost/blob/main/LICENSE)

Nocalhost is cloud native development environment. Check [https://nocalhost.dev](https://nocalhost.dev) for more details.

The term "Nocalhost" originates from No localhost.

Its vision is that in the cloud era, developers use remote cloud native development environments to complete development instead of configuring developing, debugging and testing environments on local computers.

You can use Nocalhost to:

- deploy complex microservice applications to cloud environment.
- develop components(services) quickly with a prepared configuration.
- share development environment within teams. 
- accelerate the feedback loop of "coding-building-running--debugging-testing-coding".


## Why we make Nocalhost?

As microservices become more and more popular and the number of microservices increases, `containerization` technology becomes a good solution to standardize the runtime environment.

Kubernetes is a typical microservice runtime solution. But developers who build application based on Kubernetes are facing painful problems. Such as:


- In order to develop a certain microservice, it is necessary to start the entire environment and all microservices, which requires high performance of local resources, poor experience and high cost;
- Developers often only focus on the services they are responsible for. With the continuous iteration of services and configurations, it is more and more difficult for the machine to start the `new` and `complete` development environment;
- Every time the code changes, the process of build image -> push image -> pull image -> restart application (Pod) is required, the feedback loop of development is extremely long;
- When two or more developers are involved in remote collaboration and joint debugging, they need a flatten network(VPN is too complicated to configure).

## How to solve?

Based on Kubernetes, Nocalhost provides several features:
* Quickly create an application development environment based on Kubernetes Namespace isolation for each team member, which promises development and debugging will not affect each other;
* Cloud native experience microservice development and debugging: No needs to start any microservices on local machine. Any code changes will be synchronized immediately to the remote Pod without rebuilding images.
* Starting services orderly. Such as: "Mysql (UP & Init) -> RabbitMQ (UP) -> Server A (UP) —> Server B (UP)"

# Nocalhost components

## Nocalhost Server

- Nocalhost Api
- Nocalhost Web

Nocalhost Server manages applications, clusters, users and authorizations.

## Nocalhost Dep

Nocalhost Dep is an Agent running in Kubernetes which controls the starting order of services.

## nhctl

nhctl is a command line tool running locally. It controls status of applications and services.

## IDE Plugin

Focusing on better experiences, Nocalhost provides developers with IDE Plugins to connect Cloud and local IDE directly.

- [Visual Studio Code Extension](https://marketplace.visualstudio.com/items?itemName=nocalhost.nocalhost)
- [IntelliJ Plugin](https://plugins.jetbrains.com/plugin/16058-nocalhost)

# Quick Start

[https://nocalhost.dev/getting-started/](https://nocalhost.dev/getting-started/)

# Developing

## build nhctl

```
make nhctl
```

## build Api Server

```
make api
```

## build nocalhost-dep

```
make nocalhost-dep
```

## Api Docs

```
swag init -g cmd/nocalhost-api/nocalhost-api.go
```

Then you can visit: http://127.0.0.1:8080/swagger/index.html


# Contribution

- Code Of Conduct: https://github.com/cncf/foundation/blob/master/code-of-conduct.md
- Any suggestions could be committed as a GitHub Issue: https://github.com/nocalhost/nocalhost/issues
- Pull Requests are welcomed: https://github.com/nocalhost/nocalhost/pulls


# 🔥We're Hiring

If you are interested with Nocalhost, please send resume to email: wangweimax@coding.net