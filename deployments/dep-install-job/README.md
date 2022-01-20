# nocalhost-dep 部署备忘
1. 创建 nocalhost-reserved 命名空间，作为 nocalhost 系统命名空间（服务端添加集群时处理）
2. 使用用户 admin 用户权限的 kubeconfig 创建 example/kube-confiamap-example.yaml configmap，用于凭据挂载（服务端添加集群时处理）
3. 创建用户态 namespace，并标记 label env=nocalhost ，用于自动注入（授权用户时处理）

# 部署原理
1. installer-job 会启动一个 Job，在 Pod 内执行以下操作
2. 生成 admission webhook 的证书，并写入 configMap 再部署
3. 替换 webhook/mutating-webhook.yaml CA_BUNDLE，对应 kubeconfig 的 `certificate-authority-data`，生成 mutating-webhook-ca-bundle.yaml
4. 部署 webhook/mutating-webhook-ca-bundle.yaml、webhook/sidecar-configmap.yaml、webhook/deployment.yaml 和 webhook/service.yaml

# 构建
## nocalhost-dep
需要从项目根目录构建，并向 docker 手动传递上下文：
```
make dep-docker
```
对应镜像：nocalhost-docker.pkg.coding.net/nocalhost/public/nocalhost-dep

## dep-installer

```
make dep-installer-job-docker
```

对应镜像：nocalhost-docker.pkg.coding.net/nocalhost/public/dep-installer-job