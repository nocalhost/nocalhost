module nocalhost

go 1.16

// require k8s.io/kubernetes v1.16.10

require (
	github.com/alecthomas/template v0.0.0-20190718012654-fb15b899a751
	github.com/alicebob/gopher-json v0.0.0-20200520072559-a9ecdc9d1d3a // indirect
	github.com/alicebob/miniredis v2.5.0+incompatible
	github.com/briandowns/spinner v1.11.1
	github.com/cheggaaa/pb/v3 v3.0.5
	github.com/derailed/tview v0.6.6
	github.com/containerd/containerd v1.5.8
	github.com/docker/libcontainer v2.2.1+incompatible
	github.com/fastly/go-utils v0.0.0-20180712184237-d95a45783239 // indirect
	github.com/fatih/color v1.7.0
	github.com/fsnotify/fsnotify v1.4.9
	github.com/gdamore/tcell/v2 v2.4.1-0.20210905002822-f057f0a857a1
	github.com/ghodss/yaml v1.0.0
	github.com/gin-contrib/pprof v1.3.0
	github.com/gin-gonic/gin v1.7.0
	github.com/go-ldap/ldap/v3 v3.4.1
	github.com/go-playground/validator/v10 v10.9.0
	github.com/go-redis/redis v6.15.9+incompatible
	github.com/golang-jwt/jwt/v4 v4.1.0
	github.com/golang/glog v1.0.0
	github.com/golang/protobuf v1.5.2
	github.com/golang/snappy v0.0.4
	github.com/gomodule/redigo v2.0.0+incompatible // indirect
	github.com/google/uuid v1.2.0
	github.com/hashicorp/golang-lru v0.5.1
	github.com/heroku/docker-registry-client v0.0.0-20190909225348-afc9e1acc3d5
	github.com/imroc/req v0.3.0
	github.com/jehiah/go-strftime v0.0.0-20171201141054-1d33003b3869 // indirect
	github.com/jinzhu/gorm v1.9.16
	github.com/lestrrat-go/file-rotatelogs v2.4.0+incompatible
	github.com/lestrrat-go/strftime v1.0.3 // indirect
	github.com/mattn/go-sqlite3 v2.0.1+incompatible // indirect
	github.com/mattn/psutil v0.0.0-20170126005127-e6c88f1e9be6
	github.com/miekg/dns v1.0.14
	github.com/milosgajdos/tenus v0.0.3
	github.com/mitchellh/go-ps v1.0.0
	github.com/moby/term v0.0.0-20210610120745-9d4ed1856297
	github.com/muesli/cache2go v0.0.0-20210519043705-f6c4b2d7bc5d
	github.com/nocalhost/remotecommand v0.0.0-20210713025133-bfbc2d3e1250
	github.com/olekukonko/tablewriter v0.0.4
	github.com/olivere/elastic/v7 v7.0.27
	github.com/opencontainers/image-spec v1.0.2 // indirect
	github.com/pkg/errors v0.9.1
	github.com/prometheus/client_golang v1.11.0
	github.com/qiniu/api.v7 v0.0.0-20190520053455-bea02cd22bf4
	github.com/rivo/tview v0.0.0-20211202162923-2a6de950f73b
	github.com/satori/go.uuid v1.2.0
	github.com/sirupsen/logrus v1.8.1
	github.com/songgao/water v0.0.0-20200317203138-2b4b6d7c09d8
	github.com/spf13/cast v1.3.1
	github.com/spf13/cobra v1.2.1
	github.com/spf13/pflag v1.0.5
	github.com/spf13/viper v1.8.1
	github.com/swaggo/gin-swagger v1.2.0
	github.com/swaggo/swag v1.6.2
	github.com/syndtr/goleveldb v1.0.0
	github.com/tebeka/strftime v0.1.5 // indirect
	github.com/tencentcloud/tencentcloud-sdk-go v1.0.148
	github.com/teris-io/shortid v0.0.0-20171029131806-771a37caa5cf
	github.com/tidwall/sjson v1.2.4
	github.com/toolkits/net v0.0.0-20160910085801-3f39ab6fe3ce
	github.com/ulikunitz/xz v0.5.8
	github.com/vmihailenco/msgpack v4.0.4+incompatible
	github.com/willf/pad v0.0.0-20200313202418-172aa767f2a4
	github.com/yuin/gopher-lua v0.0.0-20200816102855-ee81675732da // indirect
	go.uber.org/zap v1.19.0
	golang.org/x/crypto v0.0.0-20211215153901-e495a2d5b3d3
	golang.org/x/mod v0.4.2
	golang.org/x/net v0.0.0-20211215060638-4ddde0e984e9
	golang.org/x/sync v0.0.0-20210220032951-036812b2e83c
	golang.org/x/sys v0.0.0-20211216021012-1d35b9e2eb4e
	golang.org/x/text v0.3.8-0.20211105212822-18b340fc7af2
	golang.zx2c4.com/wireguard v0.0.0-20210927201915-bb745b2ea326
	golang.zx2c4.com/wireguard/windows v0.4.10
	google.golang.org/grpc v1.40.0
	google.golang.org/protobuf v1.27.1
	gopkg.in/natefinch/lumberjack.v2 v2.0.0
	gopkg.in/yaml.v3 v3.0.0-20210107192922-496545a6307b
	istio.io/api v0.0.0-20210809175348-eff556fb5d8a
	istio.io/client-go v1.11.0
	k8s.io/api v0.21.2
	k8s.io/apiextensions-apiserver v0.21.2
	k8s.io/apimachinery v0.21.2
	k8s.io/cli-runtime v0.21.0-alpha.1
	k8s.io/client-go v0.21.2
	k8s.io/kubectl v0.21.0-alpha.1
	sigs.k8s.io/kustomize v2.0.3+incompatible
	sigs.k8s.io/yaml v1.2.0
)

replace (
	github.com/go-playground/validator/v10 => github.com/nocalhost/validator/v10 v10.9.0-nocalhost
	k8s.io/api => k8s.io/api v0.0.0-20210115125903-c873f2e8ab25
	k8s.io/apimachinery => k8s.io/apimachinery v0.21.0-alpha.1.0.20210121071119-460d10991a52
	k8s.io/cli-runtime => k8s.io/cli-runtime v0.0.0-20210121193827-3659b9895efa
	k8s.io/client-go => k8s.io/client-go v0.0.0-20210121071529-b72204b2445d
	k8s.io/kubectl => k8s.io/kubectl v0.0.0-20210121234059-952f50e545b1
)
