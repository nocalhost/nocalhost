package tpl

import (
	"bytes"
	"text/template"
)

var svcTpl = `# service name
# type: string
# required
name: {{ . }}
    
# kubernetes workload, such as: deployment,job
# type: select, options: deployment/statefulset/pod/job/cronjob/daemonset case-insensitive
# required
serviceType: deployment

# git url where the source code of this service resides
# type: string
# default value: null
# required
gitUrl: "https://github.com/nocalhost/nocalhost.git"

# develop container image
# type: string
# default value: codingcorp-docker.pkg.coding.net/nocalhost/dev-images/golang:latest
# required
devContainerImage: "codingcorp-docker.pkg.coding.net/nocalhost/dev-images/golang:latest"

# dirs to sync, relative to the root dir of source code
# type: string[]
# default value: ["."]
# optional
syncDirs:
  - "."

# work dir of develop container
# type: string
# default value: "/home/nocalhost-dev"
# optional
# workDir: "/home/nocalhost-dev"

# ports which need to be forwarded
# localPort:remotePort
# type: string[]
# default value: []
# optional
# devPorts:
#   - 8080:8080
#   - :8000  # random localPort, remotePort 8000

# pod selectors which service depends on
# type: string[]
# default value: []
# optional
# dependPodsLabelSelector:
#   - "name=mariadb"
#   - "app.kubernetes.io/name=mariadb"

# job selectors which service depends on
# type: string[]
# default value: []
# optional
# dependJobsLabelSelector:
#   - "name=init-job"
#   - "app.kubernetes.io/name=init-job"
`

func GetSvcTpl(svcName string) (string, error) {
	t, err := template.New(svcName).Parse(svcTpl)
	if err != nil {
		return "", err
	}
	buf := new(bytes.Buffer)
	err = t.Execute(buf, svcName)
	if err != nil {
		return "", err
	}
	return buf.String(), nil
}
