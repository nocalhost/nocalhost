# 备忘录
1. kubectl create ns nocalhost-reserved  # nocalhost NS
2. apply kube-configmap-example.yaml   # 管理员 kubeconfig 用于挂载到 admission webhook 
3. create installer-job.yaml # create 避免 job 重名，生成 admission webhook cert
4. apply webhook/deployment.yaml && service.yaml  # 部署 admission webhook
5. apply webhook/mutating-webhook.yaml  # 部署前需要先替换 caBundle 集群根证书
6. kubectl label namespace default env=nocalhost   # 对 NS 打标签，自动注入
7. kubectl create rolebinding default-view \
     --clusterrole=view \
     --serviceaccount=default:default \
     --namespace=default  # 为 wait-for 设置 default serviceAccount 有 List 权限，否则 InitContainer 无法启动。
8. kubectl delete --all pods --namespace=ns