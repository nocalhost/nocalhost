# 前提条件
- kubeconfig 为管理员权限
- K8s 集群版本 >= v1.16

# 功能
dep 是一个 K8s 的 webhook，可在准入阶段对资源进行修改  
通过获取`configmaps`或者`annotations`中的配置，对资源进行配置注入  
`configmaps`和`annotations`配置详见：https://github.com/nocalhost/bookinfo/tree/config/example

# 将 dep chart 作为 chart 依赖安装
- helm 会使用依赖的 dep chart 启动一个 installer-job 来完成 dep 安装
- installer-job 使用了 helm pre-install hook 来先于业务应用启动，业务应用启动时会被拦截到 dep 中，根据`configmaps`和`annotations`的内容进行注入

在 Chart.yaml 文件中添加 dep chart 依赖
```
dependencies:
  - name: dep
    version: 0.1.0
    repository: https://nocalhost-helm.pkg.coding.net/nocalhost/helm
```
更新依赖
```
helm dep up <your_chart>
```
安装
```
helm install <name> <your_chart> --namespace <your_namespace> --kubeconfig <your_kubeconfig>
```

# 变量说明
**通常情况下不需要更改以下变量，直接引用 dep chart 为依赖即可**  
**注意：** 同一个集群多次 helm install 时 `dep.match.namespace.label` 下的 `key` `value` 需保持不变，建议使用默认值
``` yaml
## installer-job 镜像配置
image:
  repository: nocalhost-docker.pkg.coding.net/nocalhost/public/dep-installer-job
  tag: "latest"

## dep 配置
dep:
  # 安装 dep 的 namespace （webhook 不建议和业务应用在同一个空间）
  # webhook 权限较大，不应该和业务应用在一个空间
  # webhook 自己拦截自己可能会有问题
  # 建议使用默认值
  namespace: nocalhost-reserved
  # dep 镜像配置
  image:
    repository: nocalhost-docker.pkg.coding.net/nocalhost/public/nocalhost-dep
    tag: "latest"
  # webhook 匹配方式
  # 可选值 "namespaceLabel"、"namespaceName"
  # "namespaceLabel"：根据 namespace label 进行匹配，匹配中的 namespace 中的资源会被拦截到 dep 中进行注入
  # "namespaceName"：拦截所有 namespace 的资源到 dep 中，但只有 dep.match.namespace.name 指定的 namespace 中的资源会被 dep 注入，
  # 其它 namespace 的资源不进行注入
  # 使用 "namespaceName" 需要注意：在同一集群多次 helm install，后面 install 的 dep.match.namespace.name 会覆盖前面的，导致之前的
  # 的 dep.match.namespace.name 被覆盖，之前的 namespace 不会再被注入，
  # 强烈建议使用 "namespaceLabel"，在同一集群多次 helm install，每次都使用同样的 label 即可多 namespace 保持注入
  matchWith: "namespaceLabel"
  match:
    namespace:
      # 仅当 dep.matchWith=namespaceName 时有效
      # 如果为空，则使用默认值 {{ .Release.Namespace }}
      name: ""
      # 仅当 dep.matchWith=namespaceLabel 时有效
      # 具有 "<key> = <value>" label 的 namespace 会被注入
      label:
        # 注意，同一个集群多次 helm install 时， `key` `value` 需保持不变
        # 建议使用默认值，
        # 如果为空，则使用默认值 "env"
        key: ""
        # 如果为空，则使用默认值 "nocalhost"
        value: ""
```

# dep chart 作为依赖安装时进行配置覆盖
```
# 更改 installer-job 镜像
--set dep.image.repository=xxx
--set dep.image.tag=xxx
# 更改 dep 相关配置
## 更改 dep 镜像
--set dep.dep.image.repository=xxx
--set dep.dep.image.tag=xxx
## 更改匹配 namespace 的 label
--set dep.dep.match.namespace.label.key=xxx
--set dep.dep.match.namespace.label.value=xxx
```