



## Demo 执行步骤

### 前置工作

将 kubeconfig 文件拷贝到`~/.kube/config`

创建一个 namespace 用于测试：

```shell
mkdir -p /tmp/tmp/demo20
cd /tmp/tmp/demo20
kubectl create ns demo20
kubectl label namespace demo20 env=nocalhost
kubectl create rolebinding default-view \
  --clusterrole=view \
  --serviceaccount=demo20:default \
  --namespace=demo20
```

将 nhctl 可执行文件拷贝到 /usr/local/bin 目录下。

### 安装应用

#### 安装 helm 应用

```shell
nhctl install -n demo20 -r bookinfo-04 -u git@e.coding.net:codingcorp/bookinfo/bookinfo-charts.git -t helm
```

参数说明：

- -n ：namespace
- -r : helm release 的名字
- -u : helm chart 文件所在的 git 地址
- -t : 应用类型

环境清理：

```shell
helm -n demo20 uninstall bookinfo-04
```



#### 安装 manifest 应用

先访问 bookinfo 网址

```shell
nhctl install -u git@e.coding.net:codingcorp/bookinfo/bookinfo-manifest.git -d deployment -t manifest --pre-install bookinfo-manifest/pre-install.yaml -n demo20
```

pre-install.yaml 是前置定义

依赖管理

### 开发调试

#### details

Replace : 

```shell
nhctl debug  start -d details-v1  -l ruby  -n demo20 -i codingcorp-docker.pkg.coding.net/nocalhost/public/share-container-ruby:v2
```

```shell
nhctl port-forward -d details-v1 -n demo20  -l 12345 -r 22

ssh root@127.0.0.1 -p 12345
```

```shell
nhctl sync -l cccc -r /opt/microservicess -p 12345

ruby details.rb 9080

rdebug-ide details.rb 9080

kubectl port-forward pod/details-v1-f95c5cdc-dhlt5 -n demo20 12347:1234
```



```shell
nhctl debug end -d  details-v1  -n demo20
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