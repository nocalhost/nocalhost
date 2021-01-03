/*
Copyright 2020 The Nocalhost Authors.
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
    http://www.apache.org/licenses/LICENSE-2.0
Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package tpl

import (
	"bytes"
	"text/template"

	"github.com/pkg/errors"
)

var applicationTpl = `# application name
# type: string(dns1123)
# default value: null
# required
# uniq
name: nocalhost

# appplication resource type
# type: select，options：helmGit/helmRepo/rawManifest/kustomize
# default value: null
# required
manifestType: helmGit 

# Manifest resources path(relative to the root directory)
# type: string[]
# default value: ["."]
# required
resourcePath: ["deployments/chart"]

# run job before install application
# type: object[]
# default value: []
# optional
onPreInstall:
  # type: string
  # default value: null
  # required
  - path: "job-1.yaml"
    # type: integer
    # default value: 0
    # optional
    priority: -1
  - path: "job-2.yaml"
    priority: 5

`

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
		return "", errors.Wrap(err, "")
	}
	buf := new(bytes.Buffer)
	err = t.Execute(buf, svcName)
	if err != nil {
		return "", errors.Wrap(err, "")
	}
	return buf.String(), nil
}

func ConbineTpl() string {
	return applicationTpl + svcTpl
}
