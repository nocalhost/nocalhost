/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package testcase

import (
	"context"
	"github.com/pkg/errors"
	"io/ioutil"
	"nocalhost/internal/nhctl/common/base"
	"nocalhost/internal/nhctl/controller"
	"nocalhost/pkg/nhctl/log"
	"nocalhost/test/runner"
	"os"
	"sigs.k8s.io/yaml"
)

func Upgrade(nhctl runner.Client) error {
	cmd := nhctl.GetNhctl().Command(
		context.Background(), "upgrade",
		"bookinfo",
		"-u",
		"https://github.com/nocalhost/bookinfo.git",
		"--resource-path",
		"manifest/templates",
	)
	return runner.Runner.RunWithCheckResult(nhctl.SuiteName(), cmd)
}

func Config(nhctl runner.Client) error {
	cmd := nhctl.GetNhctl().Command(context.Background(), "config", "get", "bookinfo")
	return runner.Runner.RunWithCheckResult(nhctl.SuiteName(), cmd)
}

func SyncStatus(nhctl runner.Client, module string) error {
	return SyncStatusT(nhctl, module, "deployment")
}

func SyncStatusT(nhctl runner.Client, module, moduleType string) error {
	cmd := nhctl.GetNhctl().Command(context.Background(), "sync-status", "bookinfo", "-d", module, "-t", moduleType)
	return runner.Runner.RunWithCheckResult(nhctl.SuiteName(), cmd)
}

func List(nhctl runner.Client) error {
	cmd := nhctl.GetNhctl().Command(context.Background(), "list", "bookinfo")
	return runner.Runner.RunWithCheckResult(nhctl.SuiteName(), cmd)
}

func Get(nhctl runner.Client, types, appName string, checker func(string2 string) error) error {
	args := []string{types}
	if appName != "" {
		args = append(args, "-a", appName)
	}
	cmd := nhctl.GetNhctl().Command(context.Background(), "get", args...)
	stdout, stderr, err := runner.Runner.Run(nhctl.SuiteName(), cmd)
	if err != nil {
		return err
	}
	if stderr != "" {
		return errors.New(stderr)
	}
	return checker(stdout)
}

func Db(nhctl runner.Client) error {
	cmd := nhctl.GetNhctl().Command(context.Background(), "db", "size", "--app", "bookinfo")
	return runner.Runner.RunWithCheckResult(nhctl.SuiteName(), cmd)
}

func Pvc(nhctl runner.Client) error {
	cmd := nhctl.GetNhctl().Command(context.Background(), "pvc", "list")
	return runner.Runner.RunWithCheckResult(nhctl.SuiteName(), cmd)
}

func NhctlVersion(nhctl *runner.CLI) error {
	cmd := nhctl.Command(context.Background(), "version")
	stdout, stderr, err := runner.Runner.RunWithRollingOutWithChecker(nhctl.SuitName(), cmd, nil)
	return runner.Runner.CheckResult(cmd, stdout, stderr, err)
}

func Apply(nhctl runner.Client) error {
	content := `{
	"apiVersion": "v1",
	"kind": "Service",
	"metadata": {
		"name": "reviews",
	"labels": {
		"app": "reviews",
		"service": "reviews",
		"apply": "apply"
		}
	},
	"spec": {
	"ports": [
		{
		"port": 9080,
		"name": "http"
		}
	],
	"selector": {
		"app": "reviews"
		}
	}
}`
	b, _ := yaml.JSONToYAML([]byte(content))
	f, _ := ioutil.TempFile("", "*apply.yaml")
	_, _ = f.Write(b)
	_ = f.Sync()
	defer f.Close()

	cmd := nhctl.GetNhctl().Command(context.Background(), "apply", "bookinfo", f.Name())
	stdout, stderr, err := runner.Runner.RunWithRollingOutWithChecker(nhctl.SuiteName(), cmd, nil)
	return runner.Runner.CheckResult(cmd, stdout, stderr, err)
}

func RemoveSyncthingPidFile(nhctl runner.Client, module string) error {
	defer func() {
		if err := recover(); err != nil {
			log.Warnf("remove syncthing pid file failed, err: %v", err)
		}
	}()
	c := &controller.Controller{
		NameSpace: nhctl.GetKubectl().Namespace,
		AppName:   "bookinfo",
		Name:      module,
		Type:      base.Deployment,
	}
	return os.Remove(c.GetSyncThingPidFile())
}
