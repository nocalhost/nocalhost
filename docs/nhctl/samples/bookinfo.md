



## Demo 执行步骤

### 前置工作

将 kubeconfig 文件拷贝到`~/.kube/config`

创建一个 namespace 用于测试：

```shell
mkdir -p /tmp/tmp/demo20
cd /tmp/tmp/demo20
kubectl create ns demo20
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





```shell
#!/bin/bash

# ./nhctl install -n demo -r bookinfo-04 -u git@e.coding.net:codingcorp/bookinfo/bookinfo-charts.git -k ~/.kube/admin-config # -t helm
./nhctl install -u git@e.coding.net:codingcorp/bookinfo/bookinfo-manifest.git -d deployment -k ~/.kube/admin-config -t manifest --pre-install items.yaml -n demo4
./nhctl debug  start -d details-v1  -l ruby  -n demo --kubeconfig /Users/xinxinhuang/.kube/admin-config
./nhctl port-forward -d details-v1 -n demo  -l 12345 -r 22 -k ~/.kube/admin-config
./nhctl sync -l share1 -r /opt/microservicess -p 12345
./nhctl debug end -d  details-v1  -n demo --kubeconfig  /Users/xinxinhuang/.kube/admin-config
```