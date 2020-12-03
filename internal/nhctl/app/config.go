package app

import "errors"

// # application name
// # type: string(dns1123)
// # default value: null
// # required
// # nhctl param: [NAME]
// # uniq
// name: nocalhost
//
// # application menifest install type
// # type: select，options：helm/mainfest/kustomize
// # default value: null
// # required
// # nhctl param: --type,-t
// manifestType: helm
//
// # application mainfest file or chart file relative path, manifest will save multi path
// # type: string[]
// # default value: ["."]
// # required
// resourcePath: ["deployments/chart"]
//
// # application resource limit (TODO)
// # type: boolean
// # default value: false
// # optional
// # nhctl param: TODO
// minimalInstall: false
//
// # before install job
// # type: object[]
// # default value: []
// # optional
// onPreInstall:
//
// # after install jbo (TODO)
// # type: object[]
// # default value: []
// # optional
// onPostInstall:
//
// # befaore uninstall (TODO)
// # type: object[]
// # default value: []
// # optional
// onPreUninstall:
//
// # after uninstall (TODO)
// # type: object[]
// # default value: []
// # optional
// onPostUninstall:
//
// # service list
// # type: object[]
// # default value: []
// # optional
// services:

type Config struct {
	Name            string     `json:"name" yaml:"name"`
	ManifestType    string     `json:"manifestType" yaml:"manifestType"`
	ResourcePath    []string   `json:"resourcePath" yaml:"resourcePath"`
	MinimalInstall  bool       `json:"minimalInstall" yaml:"minimalInstall"`
	OnPreInstall    []*Hook    `json:"onPreInstall" yaml:"onPreInstall"`
	OnPostInstall   []*Hook    `json:"onPostInstall" yaml:"onPostInstall"`
	OnPreUninstall  []*Hook    `json:"onPreUninstall" yaml:"onPreUninstall"`
	OnPostUninstall []*Hook    `json:"onPostUninstall" yaml:"onPostUninstall"`
	Service         []*Service `json:"service" yaml:"service"`
}

// # Job yaml 相对于源码目录的位置
// # type: string
// # default value: null
// # required
// - path: "job-1.yaml"
//
//   # Job yaml 相对于源码目录的位置（预留，以后支持）
//   # type: string
//   # default value: null
//   # required
//   name: xxx-job
//
//   # job 执行的先后顺序，小的先执行
//   # type: integer
//   # default value: 0
//   # optional
//   priority: -1
type Hook struct {
	Path     string `json:"path" yaml:"path"`
	Name     string `json:"name" yaml:"name"`
	Priority int64  `json:"priority" yaml:"priority"`
}

//     # 服务名称，与服务名称的正则表达式两者只能配置一个，如果同时配置，则 name 属性生效，nameRegex 属性不生效
//     # type: string
//     # default value: null
//     # optional
//   - name: service1
//     # 服务名称的正则表达式
//     # type: string
//     # default value: null
//     # optional
//     nameRegex: .*-mariadb
//     # 服务对应的 Kubernetes 的工作负载类型
//     # type: select, options: deployment/statefulset/pod/job/cronjob/daemonset 大小写不敏感
//     # default value: deployment
//     # required
//     serviceType: deployment
//     # 服务对应的源代码的 Git Clone 地址
//     # type: string
//     # default value: null
//     # required
//     gitUrl: "https://github.com/nocalhost/nocalhost.git"
//
//     # 开发此服务的开发环境的镜像
//     # type: string
//     # default value: codingcorp.coding.net/xxxx/go:latest
//     # required
//     devContainerImage: "codingcorp.coding.net/xxxx/go:latest"
//
//     # 开发此服务的开发环境的镜像的 shell
//     # type: string
//     # default value: "/bin/sh"
//     # optional
//     devContainerShell: "bash"
//
//     # 开发模式中文件同步类型（预留，以后支持）
//     # "send" 为单向同步到容器内，"sendreceive" 为双向同步
//     # type: select，send/sendreceive
//     # default value: "send"
//     # optional
//     syncType: send
//
//     # 开发模式中同步的文件目录列表
//     # type: string[]
//     # default value: ["."]
//     # optional
//     syncDirs:
//       - "./src"
//       - "./pkg/fff"
//
//     # 开发模式中忽略的文件目录列表
//     # type: string[]
//     # default value: []
//     # optional
//     ignoreDirs:
//       - ".git"
//       - "./build"
//
//     # 开发模式需要转发端口到本地
//     # localPort:remotePort
//     # type: string[]
//     # default value: []
//     # optional
//     devPort:
//       - 8080:8080
//       - :8000  # random localPort, remotePort 8000
//
//     # 服务启动所依赖的 pod 标签选择器(此服务会等待标签选择器选中的 pod 启动完毕才启动)
//     # type: string[]
//     # default value: []
//     # optional
//     dependPodsLabelSelector:
//       - "name=mariadb"
//       - "app.kubernetes.io/name=mariadb"
//
//     # 服务启动所依赖的 pod 标签选择器(此服务会等待标签选择器选中的 job 执行完毕才启动)
//     # type: string[]
//     # default value: []
//     # optional
//     dependJobsLabelSelector:
//       - "name=init-job"
//       - "app.kubernetes.io/name=init-job"
//
//     # 开发容器的工作目录（源码会同步到这个工作目录，此工作目录会使用 PV 持久化，方便后续持续开发）
//     # type: string
//     # default value: "/home/nocalhost-dev"
//     # optional
//     workDir: "/home/nocalhost-dev"
//
//     # 开发容器的持久化存储目录（这个目录下的内容会被以 PV 形式保存，方便后续开发复用）（预留，以后支持）
//     # type: string[]
//     # default value: ["/home/nocalhost-dev"]
//     # optional
//     persistentVolumeDir:
//       - "/home/nocalhost-dev"
//
//     # 服务的构建命令（用于在代码变动后执行编译）
//     # type: string
//     # default value: ""
//     # optional
//     buildCommand: "./gradlew package"
//
//     # 服务启动命令
//     # type: string
//     # default value: ""
//     # optional
//     runCommand: "./gradlew bootRun"
//
//     # 服务调试启动命令
//     # type: string
//     # default value: ""
//     # optional
//     debugCommand: "./gradlew bootRun --debug-jvm"
//
//     # 热加载服务启动命令
//     # type: string
//     # default value: ""
//     # optional
//     hotReloadRunCommand: "kill `ps -ef|grep -i gradlew| grep -v grep| awk '{print $2}'`; gradlew bootRun"
//
//     # 热加载服务调试启动命令
//     # type: string
//     # default value: ""
//     # optional
//     hotReloadDebugCommand: "kill `ps -ef|grep -i gradlew| grep -v grep| awk '{print $2}'`; gradlew bootRun --debug-jvm"
//
//     # 服务调试端口
//     # type: string
//     # default value: ""
//     # optional
//     remoteDebugPort: 5005
//
//     # 使用 VSCode 的 dev-container 机制作为开发基础环境（预留，未来实现）
//     # type: string
//     # default value: ""
//     # optional
//     useDevContainer: false
type Service struct {
	Name                    string   `json:"name" yaml:"name"`
	NameRegex               string   `json:"nameRegex" yaml:"nameRegex"`
	ServiceType             string   `json:"serviceType" yaml:"serviceType"`
	GitUrl                  string   `json:"gitUrl" yaml:"gitUrl"`
	DevContainerImage       string   `json:"devContainerImage" yaml:"devContainerImage"`
	DevContainerShell       string   `json:"devContainerShell" yaml:"devContainerShell"`
	SyncType                string   `json:"syncType" yaml:"syncType"`
	SyncDirs                []string `json:"syncDirs" yaml:"syncDirs"`
	IgnoreDirs              []string `json:"ignoreDirs" yaml:"ignoreDirs"`
	DevPort                 []string `json:"devPort" yaml:"devPort"`
	DependPodsLabelSelector []string `json:"dependPodsLabelSelector" yaml:"dependPodsLabelSelector"`
	DependJobsLabelSelector []string `json:"dependJobsLabelSelector" yaml:"dependJobsLabelSelector"`
	WorkDir                 string   `json:"workDir" yaml:"workDir"`
	PersistentVolumeDir     []string `json:"persistentVolumeDir" yaml:"persistentVolumeDir"`
	BuildCommand            string   `json:"buildCommand" yaml:"buildCommand"`
	RunCommand              string   `json:"runCommand" yaml:"runCommand"`
	DebugCommand            string   `json:"debugCommand" yaml:"debugCommand"`
	HotReloadRunCommand     string   `json:"hotReloadRunCommand" yaml:"hotReloadRunCommand"`
	HotReloadDebugCommand   string   `json:"hotReloadDebugCommand" yaml:"hotReloadDebugCommand"`
	RemoteDebugPort         string   `json:"remoteDebugPort" yaml:"remoteDebugPort"`
	UseDevContainer         string   `json:"useDevContainer" yaml:"useDevContainer"`
}

func (c *Config) CheckValid() error {
	if len(c.Name) == 0 {
		return errors.New("Application name is missing")
	}
	if len(c.ManifestType) == 0 {
		return errors.New("Application manifestType is missing")
	}
	if len(c.ResourcePath) == 0 {
		return errors.New("Application resourcePath is missing")
	}
	preInstall := c.OnPreInstall
	if preInstall != nil && len(preInstall) > 0 {
		for _, hook := range preInstall {
			if len(hook.Name) == 0 {
				return errors.New("Application preInstall item's name is missing")
			}
			if len(hook.Path) == 0 {
				return errors.New("Application preInstall item's path is missing")
			}
		}
	}
	postInstall := c.OnPostInstall
	if postInstall != nil && len(postInstall) > 0 {
		for _, hook := range postInstall {
			if len(hook.Name) == 0 {
				return errors.New("Application postInstall item's name is missing")
			}
			if len(hook.Path) == 0 {
				return errors.New("Application postInstall item's path is missing")
			}
		}
	}
	preUninstall := c.OnPreUninstall
	if preUninstall != nil && len(preUninstall) > 0 {
		for _, hook := range preUninstall {
			if len(hook.Name) == 0 {
				return errors.New("Application preUninstall item's name is missing")
			}
			if len(hook.Path) == 0 {
				return errors.New("Application preUninstall item's path is missing")
			}
		}
	}
	postUninstall := c.OnPostUninstall
	if postUninstall != nil && len(postUninstall) > 0 {
		for _, hook := range postUninstall {
			if len(hook.Name) == 0 {
				return errors.New("Application postUninstall item's name is missing")
			}
			if len(hook.Path) == 0 {
				return errors.New("Application postUninstall item's path is missing")
			}
		}
	}
	services := c.Service
	if services != nil && len(services) > 0 {
		for _, s := range services {
			if len(s.ServiceType) == 0 {
				return errors.New("Application services serviceType is missing")
			}
			if len(s.GitUrl) == 0 {
				return errors.New("Application services gitUrl is missing")
			}
			if len(s.DevContainerImage) == 0 {
				return errors.New("Application services devContainerImage is missing")
			}
		}
	}
	return nil
}
