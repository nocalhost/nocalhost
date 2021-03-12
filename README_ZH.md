# Nocalhost

Nocalhost 是云原生开发环境。

Nocalhost 一词源于 No localhost，其愿景是开发者在云时代实现无须在本地电脑配置开发、调试、测试环境，直接使用远端的云原生开发环境完成开发。

你可以使用 Nocalhost:

- 一键配置复杂的微服务应用到云原生开发环境
- 在预配置好的情况下，快速开发某个微服务组件
- 与协作者无缝共享开发环境
- 快速的“编码->构建->运行->调试”反馈循环


## 为什么要做 Nocalhost 项目？
随着微服务越来越流行，应用的微服务数量越来越多，团队会使用 `容器化` 技术来屏蔽微服务的环境差异。

而对于使用 Kubernetes 作为基础环境的系统来说，他们的开发和调试变得越来越困难，主要体现在以下几点：

- 为了对某一个微服务进行开发，需要启动整个环境以及所有的微服务，对本机资源性能要求高，体验差且成本高昂；
- 开发人员往往只专注于自己负责的服务，随着服务和配置的不断迭代，本机启动`最新`且`完整`的开发环境越来越困难；
- 每次代码改动，都需要 build 镜像 -> 推送镜像 -> 拉取镜像 -> 重建应用（Pod） 的流程，开发的反馈循环极慢；
- 当涉及两人或更多的人远程协作，联合调试时，本地开发环境更是只能依赖 VPN 等方式实现，配置复杂

## 如何解决？
Nocalhost 是云原生开发环境，现阶段在 Kubernetes 的基础上，为用户提供以下能力：
* 为每一位团队成员快速创建基于 Kubernetes Namespace 隔离的应用开发环境，开发调试互不影响；
* 云原生体验的微服务开发和调试：远端启动应用环境后，本机不再需要启动任何微服务，一切开发基于远端的 K8S 开发环境，无需重建 Docker 镜像，任何代码改动的影响将立即同步到对应远端的 Pod。
* 以 Sidecar 的方式解决服务启动依赖问题和服务启动顺序的编排，例如实现以下启动顺序：Mysql (UP & Init) -> RabbitMQ (UP) -> Server A (UP) —> Server B (UP)

## 愿景
Nocalhost 的最终目标是实现极致的云开发体验：

* 在 IDE 中登录 Nocalhost 自动获取有权开发的应用和云资源；
* 选择应用并部署独立的云开发环境；
* 部署完成，选择要开发的微服务组件，点击我要开发按钮；
* 自动检出代码，并编辑器内的改动自动同步到远端对应微服务容器内；
* 远端容器自动运行新的代码，改动实时生效；
* 如需调试，点击调试按钮，自动与远端建立调试通道，接收调试信息；
* 开发结束，可选销毁或者重置环境。

# Nocalhost 组成
## Web 端
Web 端提供应用管理、应用授权、集群管理、用户管理和应用-集群授权等功能。

## nhctl
nhctl 是运行在开发者本地的客户端，主要提供本地和远端云资源的交互能力，先阶段只具备操作 Kubernetes 集群的能力。

## IDE 插件

Nocalhost 以开发者体验为中心，会把与开发者相关 nhctl，登录认证等能力都封装到 IDE 插件中，开发者只要打开 IDE，即可畅享云原生开发。

- [Visual Studio Code 插件](https://marketplace.visualstudio.com/items?itemName=nocalhost.nocalhost)
- [IntelliJ 系列插件](https://plugins.jetbrains.com/plugin/16058-nocalhost)

插件提供对远程开发环境的工作负载展示、进入开发环境，克隆项目代码，调试等能力。

# 安装和开始使用

[https://nocalhost.dev/getting-started/](https://nocalhost.dev/getting-started/)

# 开发

## 构建 nhctl

```
make nhctl
```

## 构建 Api Server

```
make api
```

## 构建 nocalhost-dep

```
make nocalhost-dep
```

## 生成 API 文档
```
swag init -g cmd/nocalhost-api/nocalhost-api.go
```
访问地址：http://127.0.0.1:8080/swagger/index.html

# 参与贡献指南

- 贡献者行为准则：https://github.com/cncf/foundation/blob/master/code-of-conduct.md
- 对 Nocalhost 有任何意见和建议，请提交 GitHub Issue: https://github.com/nocalhost/nocalhost/issues
- 您可以通过提交 Pull Request 来参与社区贡献：https://github.com/nocalhost/nocalhost/pulls

# 🔥招聘

如果你对 Nocalhost 感兴趣，欢迎投递简历至：wangweimax@coding.net（接受远程）