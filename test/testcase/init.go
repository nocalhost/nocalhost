/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
*/

package testcase

import (
	"context"
	"fmt"
	"github.com/imroc/req"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"
	"io/ioutil"
	"nocalhost/internal/nhctl/app"
	"nocalhost/internal/nhctl/profile"
	"nocalhost/internal/nhctl/request"
	"nocalhost/internal/nhctl/utils"
	"nocalhost/pkg/nhctl/clientgoutils"
	"nocalhost/pkg/nhctl/log"
	"nocalhost/pkg/nhctl/tools"
	"nocalhost/pkg/nocalhost-api/app/api/v1/service_account"
	"nocalhost/test/runner"
	"nocalhost/test/util"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

var StatusChan = make(chan int32, 1)

func GetVersion() (lastVersion string, currentVersion string) {
	commitId := os.Getenv(util.CommitId)
	var tags []string
	if len(os.Getenv(util.Tag)) != 0 {
		tags = strings.Split(strings.TrimSuffix(os.Getenv(util.Tag), "\n"), " ")
	}
	if commitId == "" && len(tags) == 0 {
		panic(fmt.Sprintf("test case failed, can not found any version, commit_id: %v, tag: %v", commitId, tags))
	}
	if len(tags) >= 2 {
		lastVersion = tags[0]
		currentVersion = tags[1]
	} else if len(tags) == 1 {
		currentVersion = tags[0]
	} else {
		currentVersion = commitId
	}
	log.Infof("version info, lastVersion: %s, currentVersion: %s", lastVersion, currentVersion)
	return
}

func InstallNhctl(version string) error {
	var name string
	var needChmod bool
	if runtime.GOOS == "darwin" {
		name = "nhctl-darwin-amd64"
		needChmod = true
	} else if runtime.GOOS == "windows" {
		name = "nhctl-windows-amd64.exe"
		needChmod = false
	} else {
		name = "nhctl-linux-amd64"
		needChmod = true
	}
	str := "curl --fail -s -L \"https://codingcorp-generic.pkg.coding.net/nocalhost/nhctl/%s?version=%s\" -o %s"
	cmd := exec.Command("sh", "-c", fmt.Sprintf(str, name, version, utils.GetNhctlBinName()))
	if utils.IsWindows() {
		delCmd := exec.Command("sh", "-c", fmt.Sprintf("rm %s", utils.GetNhctlBinName()))
		if _, _, err := runner.Runner.RunWithRollingOutWithChecker(delCmd, nil); err != nil {
			log.Error(err)
		}
	}
	if err := runner.Runner.RunWithCheckResult(cmd); err != nil {
		return err
	}
	// unix and linux needs to add x permission
	if needChmod {
		cmd = exec.Command("sh", "-c", "chmod +x nhctl")
		if err := runner.Runner.RunWithCheckResult(cmd); err != nil {
			return err
		}
		cmd = exec.Command("sh", "-c", "sudo mv ./nhctl /usr/local/bin/nhctl")
		if err := runner.Runner.RunWithCheckResult(cmd); err != nil {
			return err
		}
	}
	return nil
}

func Init(nhctl *runner.CLI) error {
	cmd := nhctl.CommandWithNamespace(
		context.Background(),
		"init", "nocalhost", "demo", "-p", "7000", "--force",
	)
	log.Infof("Running command: %s", cmd.Args)
	go func() {
		_, _, err := runner.Runner.RunWithRollingOutWithChecker(
			cmd,
			func(s string) bool {
				if strings.Contains(s, "Nocalhost init completed") {
					StatusChan <- 0
					return true
				}
				return false
			},
		)
		if err != nil {
			StatusChan <- 1
		}
	}()

	if i := <-StatusChan; i != 0 {
		return errors.New("Init nocalhost occurs error, exiting")
	}
	log.Infof("init successfully")
	return nil
}

func StatusCheck(nhctl runner.Client, moduleName string) error {
	retryTimes := 10
	var ok bool
	for i := 0; i < retryTimes; i++ {
		time.Sleep(time.Second * 2)
		cmd := nhctl.GetNhctl().Command(context.Background(), "describe", "bookinfo", "-d", moduleName)
		stdout, stderr, err := runner.Runner.Run(cmd)
		if err != nil {
			log.Infof("Run command: %s, error: %v, stdout: %s, stderr: %s, retry", cmd.Args, err, stdout, stderr)
			continue
		}
		service := profile.SvcProfileV2{}
		_ = yaml.Unmarshal([]byte(stdout), &service)
		if !service.Developing {
			log.Info("test case failed, should be in developing, retry")
			continue
		}
		if !service.PortForwarded {
			log.Info("test case failed, should be in port forwarding, retry")
			continue
		}
		if !service.Syncing {
			log.Info("test case failed, should be in synchronizing, retry")
			continue
		}
		ok = true
		break
	}
	if !ok {
		return errors.New("test case failed, status check not pass")
	}
	return nil
}

func GetKubeconfig(ns, kubeconfig string) (string, error) {
	client, err := clientgoutils.NewClientGoUtils(kubeconfig, ns)
	log.Infof("kubeconfig %s", kubeconfig)
	if err != nil || client == nil {
		return "", errors.Errorf("new go client fail, or check you kubeconfig, err: %v", err)
	}
	kubectl, err := tools.CheckThirdPartyCLI()
	if err != nil {
		return "", errors.Errorf("check kubectl error, err: %v", err)
	}
	res := request.NewReq("", kubeconfig, kubectl, ns, 7000)
	res.ExposeService()
	res.Login(app.DefaultInitAdminUserName, app.DefaultInitPassword)

	header := req.Header{"Accept": "application/json", "Authorization": "Bearer " + res.AuthToken, "content-type": "text/plain"}

	retryTimes := 20
	var config string
	for i := 0; i < retryTimes; i++ {

		resp, _ := req.New().Post(
			res.BaseUrl+util.WebDevSpace, header,
			`{"cluster_id":1,"space_name":"suuuper","user_id":1,"cluster_admin":1,"cpu":0,"memory":0,"isLimit":false,"space_resource_limit":{}}`,
		)
		log.Infof(resp.String())

		time.Sleep(time.Second * 2)
		r, err := req.New().Get(res.BaseUrl+util.WebServerServiceAccountApi, header)
		if err != nil {
			log.Infof("get kubeconfig error, err: %v, response: %v, retrying", err, r)
			continue
		}
		re := Response{}
		err = r.ToJSON(&re)
		if re.Code != 0 || len(re.Data) == 0 || re.Data[0] == nil || re.Data[0].KubeConfig == "" {
			toString, _ := r.ToString()
			log.Infof("get kubeconfig response error, response: %v, string: %s, retrying", re, toString)
			continue
		}
		config = re.Data[0].KubeConfig
		break
	}
	if config == "" {
		return "", errors.New("Can't not get kubeconfig from webserver, please check your code")
	}
	f, _ := ioutil.TempFile("", "*newkubeconfig")
	_, _ = f.WriteString(config)
	_ = f.Sync()
	return f.Name(), nil
}

type Response struct {
	Code    int                                    `json:"code"`
	Message string                                 `json:"message"`
	Data    []*service_account.ServiceAccountModel `json:"data"`
}
