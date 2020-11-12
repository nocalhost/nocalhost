## Bookinfo 示例程序

### 编译

```shell
git clone https://e.coding.net/codingcorp/nocalhost/nocalhost.git
go build nocalhost/cmd/nhctl/nhctl.go
```

将 nhctl 可执行文件拷贝到 /usr/local/bin 目录下

[mutagen](https://codingcorp.coding.net/p/nocalhost/d/nocalhost-resources/git/tree/master/darwin/mutagen) 和 [mutagen-agents.tar.gz](https://codingcorp.coding.net/p/nocalhost/d/nocalhost-resources/git/tree/master/darwin/mutagen-agents.tar.gz) 也要拷到 /usr/local/bin 目录下 // 这个有点麻烦，后面看看有没有更好一点的方案

### 前置工作

将 kubeconfig 文件拷贝到`~/.kube/config`，k8s 集群应该提前安装好 webhook admission，如果没有的话服务依赖将无法处理（不影响服务启动）。

创建一个 namespace 用于测试：

```shell
NOCALHOST_NS=demo20
kubectl create ns $NOCALHOST_NS
kubectl label namespace $NOCALHOST_NS env=nocalhost
kubectl create rolebinding default-view \
  --clusterrole=view \
  --serviceaccount=$NOCALHOST_NS:default \
  --namespace=$NOCALHOST_NS
```



### 安装应用

```shell
nhctl install $APPNAME -u https://github.com/nocalhost/bookinfo.git --debug -n $NOCALHOST_NS
```

参数说明：

- -n ：namespace
- -u : git url 地址
- -t : 应用类型 (可选)
- --debug : 打印 debug 日志输出

安装完即可通过：`http://HOST_IP:30001/productpage` 访问 bookinfo 应用。

注意：确保主机的 30001 端口没有被别的程序占用

### 开发调试 details 服务

#### 替换开发镜像

替换 details 容器的镜像为开发容器镜像 : 

```shell
nhctl dev start $APPNAME -d details-v1 -n $NOCALHOST_NS
```

- -d : 指定要替换的服务对应的 deployment 的名字
- -l : 指定服务的语言类型，如 ruby, java
- -i : (可选)显式指定要替换的开发容器镜像

此时访问 bookinfo 应用，可以看到 details 服务已经无法访问。

#### 转发端口

转发本地端口到开发容器端口：

```shell
nhctl port-forward $APPNAME -d  details-v1 -n $NOCALHOST_NS 
```

- -d : 指定要替换的服务对应的 deployment 的名字
- -l : 指定要转发的本地端口
- -l : 指定开发容器的端口

该命令在端口转发过程中会一直阻塞，需要重新打开一个新的界面进行后续的操作，此时，在新打开的窗口使用以下命令可以登上开发容器：

```shell
ssh root@shared-container -p 12347
```



#### 同步文件

同步文件目录：

```shell
mkdir cccc
nhctl sync $APPNAME -d details-v1
```

- -l : 要同步的本地文件目录
- -r : 目标容器内的目录

ruby 程序源代码可以从这里下载：https://github.com/istio/istio/blob/master/samples/bookinfo/src/details/details.rb

将该文件放到本地同步目录中，同步完可在容器内执行以下命令将服务跑起来：

```shell
ruby details.rb 9080 
```

此时访问 bookinfo 应用应该能看到 details 服务已经恢复正常。

或者进行 debug ：

```shell
rdebug-ide details.rb 9080 # debug 模式运行 details 服务

# 要在本地脸上容器的 debug 端口需要将本地端口转发到容器的 debug 端口
# kubectl port-forward pod/details-v1-f95c5cdc-dhlt5 -n $NOCALHOST_NS 12347:1234
```

#### 结束开发

结束后运行以下命令可以将服务恢复为正常模式：

```shell
nhctl dev end $APPNAME -d  details-v1  -n $NOCALHOST_NS
```
