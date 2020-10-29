# 注意事项
特别留意，wait-for 容器内使用 kubectl 获取 Pod 和 Svc 的状态，会使用 `/var/run/secrets/kubernetes.io/serviceaccount` 目录的 serviceAccount。

如果当前 `default` serviceAccount 没有 List 权限，那么 InitContainer 将**无法启动**，那么需要为 default 绑定角色。

这里采用将 default 绑定 view 角色，授权 get 权限。

```
kubectl create rolebinding default-view \
  --clusterrole=view \
  --serviceaccount=default:default \
  --namespace=default
```

查看历史日志：

```
kubectl logs pod -c container --previous
```
