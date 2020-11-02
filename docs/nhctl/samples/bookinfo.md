



## Bookinfo 示例程序

### 前置工作

将 kubeconfig 文件拷贝到`~/.kube/config`，k8s 集群应该提前安装好 webhook admission。

创建一个 namespace 用于测试：

```shell
NOCALHOST_NS=demo20
kubectl create ns $NOCALHOST_NS
kubectl label namespace $NOCALHOST_NS env=nocalhost # 使 webook admission 生效
kubectl create rolebinding default-view \
  --clusterrole=view \
  --serviceaccount=demo20:default \
  --namespace=$NOCALHOST_NS
```

将 nhctl 可执行文件拷贝到 /usr/local/bin 目录下。

### 安装应用

#### 安装 helm 应用

```shell
nhctl install -n $NOCALHOST_NS -r bookinfo-04 -u git@e.coding.net:codingcorp/bookinfo/bookinfo-charts.git -t helm
```

参数说明：

- -n ：namespace
- -r : helm release 的名字
- -u : helm chart 文件所在的 git 地址
- -t : 应用类型

环境清理：

```shell
helm -n $NOCALHOST_NS uninstall bookinfo-04
```



#### 安装 manifest 应用

```shell
nhctl install -u git@e.coding.net:codingcorp/bookinfo/bookinfo-manifest.git -d deployment -t manifest --pre-install bookinfo-manifest/pre-install.yaml -n $NOCALHOST_NS
```

安装完即可通过：`http://HOST_IP:30001/productpage` 访问 bookinfo 应用。

### 开发调试 details 服务

替换 details 容器的镜像为开发容器镜像 : 

```shell
nhctl debug  start -d details-v1  -l ruby  -n $NOCALHOST_NS -i codingcorp-docker.pkg.coding.net/nocalhost/public/share-container-ruby:v2
```



```shell
nhctl port-forward -d details-v1 -n $NOCALHOST_NS  -l 12345 -r 22

ssh root@127.0.0.1 -p 12345
```

```shell
nhctl sync -l cccc -r /opt/microservicess -p 12345

# 以下命令在容器中运行
ruby details.rb 9080 # 运行 details 服务

rdebug-ide details.rb 9080 # debug 模式运行 details 服务

kubectl port-forward pod/details-v1-f95c5cdc-dhlt5 -n $NOCALHOST_NS 12347:1234
```



```shell
nhctl debug end -d  details-v1  -n $NOCALHOST_NS
```

























#### productpage

debug 

```shell
nhctl debug  start -d productpage-v1  -l ruby  -n demo20 -i codingcorp-docker.pkg.coding.net/nocalhost/public/share-container-ruby:v2
```



```shell
nhctl port-forward -d productpage-v1 -n demo20  -l 12346 -r 22

ssh root@127.0.0.1 -p 12346
```



```shell
nhctl sync -l dddd -r /opt/microservicess -p 12346
```

```shell
nhctl debug end -d  productpage-v1  -n demo20
```



#### 

```shell
#!/bin/bash

# ./nhctl install -n demo -r bookinfo-04 -u git@e.coding.net:codingcorp/bookinfo/bookinfo-charts.git -k ~/.kube/admin-config # -t helm
./nhctl install -u git@e.coding.net:codingcorp/bookinfo/bookinfo-manifest.git -d deployment -k ~/.kube/admin-config -t manifest --pre-install items.yaml -n demo4
./nhctl debug  start -d details-v1  -l ruby  -n demo --kubeconfig /Users/xinxinhuang/.kube/admin-config
./nhctl port-forward -d details-v1 -n demo  -l 12345 -r 22 -k ~/.kube/admin-config
./nhctl sync -l share1 -r /opt/microservicess -p 12345
./nhctl debug end -d  details-v1  -n demo --kubeconfig  /Users/xinxinhuang/.kube/admin-config
```