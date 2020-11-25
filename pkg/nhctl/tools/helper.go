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

package tools

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"go.uber.org/zap"
	"io"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

var logger *zap.Logger

//init
//func init() {
//    logger = zaplog.InitLogger()
//}

//check os
func CheckOS() string {
	sysType := runtime.GOOS
	//logger.Info("OS name", zap.String("type", sysType))
	switch sysType {
	case "linux":
	case "windows":
	case "darwin":
		break
	default:
		sysType = "unknown"
		break
	}
	return sysType
}

//check kubernetes version
func CheckK8s() (string, bool) {
	//osName := CheckOS()
	//if osName == "unknown" {
	//    panic("nonsupport os system. support linux, macos, windows.")
	//}
	////kubectl command
	//kubectl := osName+"/kubectl"
	//if osName == "windows" {
	//	kubectl = osName+"/kubectl.exe"
	//}

	kubectl := "kubectl"
	///home/coding-cli/
	//exec command
	command := exec.Command(kubectl, "version", "-o", "json")
	versionJson, err := command.CombinedOutput()
	if err != nil {
		fmt.Printf("cmds.Run() failed with %s\n", err)
	}
	var k8sVersion interface{}
	errVersion := json.Unmarshal(versionJson, &k8sVersion)
	if errVersion != nil {
		//logger.Error("Kubernetes api-server is not connected or Kubernetes is not installed", zap.Error(errVersion))
		fmt.Println("error: Kubernetes api-server is not connected or Kubernetes is not installed")
		return "", false
	}

	//logger.Info("Kubernetes", zap.String("version", string(versionJson)))
	version := k8sVersion.(map[string]interface{})["serverVersion"].(map[string]interface{})["gitVersion"].(string)
	return version, true
}

func ExecKubeCtlCommand(ctx context.Context, kubeconfig string, params ...string) error {
	var err error
	if kubeconfig != "" {
		params = append(params, "--kubeconfig")
		params = append(params, kubeconfig)
		_, err = ExecCommand(ctx, true, "kubectl", params...)
	} else {
		_, err = ExecCommand(ctx, true, "kubectl", params...)
	}
	return err
}

//execute command
func ExecCommand(ctx context.Context, isDisplay bool, commandName string, params ...string) (string, error) {
	var errStdout, errStderr error
	//var stdout, stderr []byte
	//execCommand := commandName

	var cmd *exec.Cmd
	if ctx == nil {
		cmd = exec.Command(commandName, params...)
	} else {
		fmt.Println("command with ctx")
		cmd = exec.CommandContext(ctx, commandName, params...)
	}
	fmt.Println(cmd.Args)
	stdoutIn, err := cmd.StdoutPipe()
	stderrIn, err2 := cmd.StderrPipe()
	if err != nil || err2 != nil {
		return "", errors.New("error: command failed to get stdout pipe")
	}
	err = cmd.Start()
	if err != nil {
		fmt.Printf("failed to start cmds, err:%v\n", err)
		return "", err
	}

	if isDisplay {
		go func() {
			_, errStdout = copyAndCapture(os.Stdout, stdoutIn)
		}()
		go func() {
			_, errStderr = copyAndCapture(os.Stderr, stderrIn)
		}()
	}
	err = cmd.Wait()
	if err != nil {
		return "", err
	}
	if errStderr != nil || errStdout != nil {
		return "", errors.New("error occur when print")
	}
	return "", nil
}

func copyAndCapture(w io.Writer, r io.Reader) ([]byte, error) {
	var out []byte
	buf := make([]byte, 1024, 1024)
	for {
		n, err := r.Read(buf[:])
		if n > 0 {
			d := buf[:n]
			out = append(out, d...)
			os.Stdout.Write(d)
		}
		if err != nil {
			// Read returns io.EOF at the end of file, which is not an error for us
			if err == io.EOF {
				err = nil
			}
			return out, err
		}
	}
	// never reached
	panic(true)
	return nil, nil
}

//Check Command
func CheckCommand(commandName string) bool {
	status := false
	toolsList := []string{
		"kubectl",
		"kubectl.exe",
		"mutagen",
		"install-k8s-master",
		"install-k8s-worker",
		"install-k8s-create-kubeconfig",
		"coding-deploy",
		"debug-service",
		"file-sync",
		"deploy-no-dependencies"}

	for _, v := range toolsList {
		if strings.Contains(commandName, v) {
			status = true
			break
		}
	}
	return status
}

//file exist check
func CheckFile(file string) bool {
	_, err := os.Stat(file)
	if err != nil && os.IsNotExist(err) {
		return false
	}
	return true
}

//check cmds exist
func CheckCmdExists(cmd string) bool {
	path, err := exec.LookPath(cmd)
	if err != nil {
		logger.Error("check command is not exists ", zap.Error(err))
		return false
	} else {
		logger.Info("check command is exists ", zap.String("command", cmd), zap.String("path", path))
		return true
	}
}
