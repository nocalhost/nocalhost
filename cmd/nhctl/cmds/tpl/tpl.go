/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package tpl

import (
	"bytes"
	"text/template"

	"github.com/pkg/errors"
)

var applicationTpl = `
configProperties:
	version: v2  # config.yaml version
	envFile: env.dev
application:
	# application name
	# type: string(dns1123)
	# default value: null
	# required
	# uniq
	name: coding-agile
	# application resource type
	# type: select，options：helmGit/helmRepo/rawManifest/kustomize
	# default value: null
	# required
	manifestType: rawManifest
	# helmRepo application's default version
	# type: string
	# default value: null
	# optional
	helmVersion: 0.0.1
	# Manifest resources path(relative to the root directory)
	# type: string[]
	# default value: ["."]
	# required
	resourcePath: ["deployments/chart"]
	ignoredPath: []
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
	# Values for helm
	helmValues:
	- key: DOMAIN
	  value: ${DOMAIN:-www.coding.com}
	- key: DEBUG
	  value: ${DEBUG:-true}
	env: 
	- name: DEBUG
	  value: ${DEBUG:-true}
	- name: DOMAIN
	  value: "www.coding.com"
	envFrom:
	  envFile: 
	  - path: dev.env
	  - path: dev.env
`

var svcTpl = `
name: e-coding
# kubernetes workload, such as: deployment,job
# type: select, options: deployment/statefulset/pod/job/cronjob/daemonset case-insensitive
# required
serviceType: deployment
dependLabelSelector: 
	# pod selectors which service depends on
	# type: string[]
	# default value: []
	# optional
	# pods: 
	#   - "name=mariadb"
	#   - "app.kubernetes.io/name=mariadb"
	# job selectors which service depends on
	# type: string[]
	# default value: []
	# optional
	# jobs:
	#   - "job-name=init-job"
	#   - "app.kubernetes.io/name=init-job"
containers:
	- name: xxx 
	  install: 
	  	env:
		- name: DEBUG
		  value: "true"
		- name: DOMAIN
		  value: "www.coding.com"
	  	envFrom: 
	  		envFile: 
			- path: dev.env
			- path: dev.env
	  	portForward:   # 安装后需要打通的端口
		- 3306:3306
	  dev:
		# git url where the source code of this service resides
		# type: string
		# default value: null
		# required
		gitUrl: xxx-job
		# develop container image
		# type: string
		# default value: nocalhost-docker.pkg.coding.net/nocalhost/dev-images/golang:latest
		# required
		image: java:8-jdk
		# The default shell of DevContainer
		# type: string
		# default value: "/bin/sh"
		# optional
		# shell: "bash"
		# work dir of develop container
		# type: string
		# default value: "/home/nocalhost-dev"
		# optional
		# workDir: "/root/nocalhost-dev"
		# storage of pv
		# type: string
		# default value: ""
		# optional
		# storageClass: "cbs"
		# resource requirements of dev container
		# resources:
		#   limits:
		#     cpu: "1"
		#     memory: 1Gi
		#   requests:
		#     cpu: "0.5"
		#     memory: 512Mi
		# Dirs to be persisted in DevContainer
		# type: string[]
		# default value: ["/home/nocalhost-dev"]
		# optional
		# persistentVolumeDirs: 
			# Dir to be persisted in DevContainer
			# type: string
			# default value: null
			# required
			# - path: "/root"
			# Capability of the dir
			# type: string
			# default value: 10Gi
			# optional
			#   capacity: 100Gi
		command: 
			# Run command of the service
			# type: string[]
			# default value: [""]
			# optional
			build: ["./gradlew", "package"]
			# Run command of the service
			# type: string[]
			# default value: [""]
			# optional
			# run: ["./gradlew", "bootRun"]
			# Debug command of the service
			# type: string[]
			# default value: [""]
			# optional
			# debug: ["./gradlew", "bootRun", "--debug-jvm"]
			# Hot-reload run command of the service
			# type: string[]
			# default value: [""]
			# optional
			hotReloadRun: ["bash", "-c", "gradlew bootRun"]
			# Hot-reload debug command of the service
			# type: string[]
			# default value: [""]
			# optional
			hotReloadDebug: ["bash", "-c", "gradlew bootRun --debug-jvm"]
		debug:
			remoteDebugPort: 5005
		useDevContainer: false
		sync:
			type: send
			# List of files and directories to be synchronized to DevContainer
			# type: string[]
			# default value: ["."]
			# optional
			filePattern:
				- "./src"
				- "./pkg/fff"
			# List of ignored files and directories to be synchronized to DevContainer
			# type: string[]
			# default value: []
			# optional
			ignoreFilePattern:
				- ".git"
				- "./build"
		env:
		- name: DEBUG
		  value: "true"
		- name: DOMAIN
		  value: "www.coding.com"
		envFrom:
		envFile:
		- path: dev.env
		- path: dev.env
		# ports which need to be forwarded
		# localPort:remotePort
		# type: string[]
		# default value: []
		# optional
		# portForward:
		# - 3306:3306
		# random localPort, remotePort 8000
		# - :8000
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

func CombineTpl() string {
	return applicationTpl + svcTpl
}
